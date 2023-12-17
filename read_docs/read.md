# Read

## NutsDB中主要结构体及功能

### 数据类型类

#### btree.go - BTree

基于`github.com/tidwall/btree`实现的一个BTree类型，BTree中的每一个数据项分别由一个key和一个record组成，包含BTree类型的常用功能接口。

### NutsDB功能依赖

#### index.go

NutsDB的内存索引。核心结构体为`index`，包含list, btree, set, sorted set四种类型所对应的索引。每个字段本质上都是一个map, key为BucketId, value为对应类型的索引。

#### entry.go

核心结构体为`Entry`，对应一条实际的数据。主要包含key, value和meta共3个字段。主要包括了对`Entry`的序列化和反序列化方法，以及`WithXXX`格式的“Setter”

#### datafile.go

核心结构体为`DataFile`，主要包括path, fileID, writeOff, ActualSize和rwManager共5个字段。主要有读取、写入、同步、关闭、释放共5个方法。

其中rwManager为一个`interface`类型，目前可选的分别是基于`FileIO`和`MMap`的RWManager。

#### bucket.go

此文件中所设计的代码为对bucket的管理，核心在于`Bucket`结构体，该结构体就是在磁盘中一个Bucket具体的抽象。

`BucketMeta`结构体定义了CRC、最后操作和一个Bucket的长度。该结构体拥有2个方法，即对其进行序列化和反序列化。

`Bucket`结构体主要包含Meta, Id, 数据结构体，Name共4个字段，分别表示一个Bucket的元数据，id, 数据结构和名称。同样的，此结构体也拥有序列化和反序列化的方法。

#### bucket_manager.go

`BucketManager`结构体，用于管理bucket，包含bucket的创建、删除、获取等功能。主要字段有：

- fd：文件描述符，用于存储bucket的元数据信息
- BucketInfoMapper: 从BucketId到Bucket的映射
- BucketIdMarker：从BucketName, DataStructure到BucketId的映射
- Gen：ID生成器

对于`BucketManager`结构体，主要包括`ExistBucket`, `GetBucket`, `GetBucketId`, `GetBucketId`等成员方法，这些方法的本质是通过给定参数在结构体中的2个映射关系中尝试获取对应的值。

此文件中存在另一个结构体，`bucketSubmitRequest`结构体用于记录对Bucket的一次提交操作，包括数据类型、Bucket名称、Bucket共3个字段。`bucketSubmitRequest`有一个成员方法`SubmitPendingBucketChange`，用于执行关于Bucket的改动，目前支持的有新建Bucket和删除Bucket。

### NutsDB功能实现

#### merge.go

具体实现了merge功能，即对数据库的文件进行合并的操作。主要函数为`merge`，主要过程为：

1. 过滤被删除的或已经过期的数据条目
2. 如果数据的key不存在，则直接将数据写入到活动文件
3. 过滤被提交的数据条目
4. 最后移除被合并的文件

主要代码流程：

1. 为db上锁，并检查db是否正在merge的过程中，如果不在merge的过程中则继续执行后续逻辑
2. 获取max file id和已经存在的file id，至少应存在2个文件，才继续进行merge
3. 当数据库使用FileIO且允许同步时，调用`activeFile.Sync()`
4. 释放已有的`activeFile`
5. 获取合并后的文件的路径并激活该文件
6. 遍历所有待合并的文件
   1. 根据给定文件id获取一个`FileRecovery`实例
   2. 遍历读取数据条目，写入到新的数据文件中
7. 移除所有待合并的文件

此文件也实现了一个`mergeWorker`函数，在打开数据库时，此函数将被调用。此函数定义了一个定时器，当定时器触发时，会调用`merge`函数。另外，当从mergeStartChannel接收到信号时调用`merge`函数并向mergeEndChannel发送信号。

## NutsDB启动流程

```go
DB struct {
    opt                     Options
    Index                   *index
    ActiveFile              *DataFile
    MaxFileID               int64
    mu                      sync.RWMutex
    KeyCount                int // total key number ,include expired, deleted, repeated.
    closed                  bool
    isMerging               bool
    fm                      *fileManager
    flock                   *flock.Flock
    commitBuffer            *bytes.Buffer
    mergeStartCh            chan struct{}
    mergeEndCh              chan error
    mergeWorkCloseCh        chan struct{}
    writeCh                 chan *request
    tm                      *ttlManager
    RecordCount             int64 // current valid record count, exclude deleted, repeated
    bm                      *BucketManager
    hintKeyAndRAMIdxModeLru *LRUCache // lru cache for HintKeyAndRAMIdxMode
}
```

以`Open`作为入口，首先对传入的DB参数进行处理，使用处理完的参数调用`open`。在`open`函数中，首先初始化一个`DB`实例，率先被初始化的字段有：

|fields|value|
|:---:|:---:|
|MaxFileID|0|
|opt|传入值|
|closed|false|
|Index|`newIndex()`|
|fm|`newFileManager()`|
|mergeStartCh|channel|
|mergeEndCh|channel|
|mergeWorkCloseCh|channel|
|writeCh||
|tm||
|hintKeyAndRAMIdxModeLru||

### `newIndex()`

初始化一个`index`结构体。`index`结构体的4个字段分别是list, btree, set, sorted set四种类型所对应的索引。

每种类型的索引为一个`map[BucketId]IdxType`类型。调用`newIndex`后会为4中类型全部赋值。

### `newFileManager()`

初始化一个`fileManager`结构体，给结构体
