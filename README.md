# FDFS

Simple Go Client of Fast DFS (Concurrent security)

[![GoDoc](https://godoc.org/github.com/beiping96/fdfs?status.svg)](https://godoc.org/github.com/beiping96/fdfs)
[![Go Report Card](https://goreportcard.com/badge/github.com/beiping96/fdfs)](https://goreportcard.com/report/github.com/beiping96/fdfs)

[FDFS](https://github.com/happyfish100/fastdfs)

## Simple Usage

``` go
package main

import (
    "github.com/beiping96/fdfs"
    "github.com/pkg/errors"
)

func main() {
    cfg := &fdfs.Config{
        Tracker: []string{"172.16.3.15:22122"},
        MaxConn: 100,
    }

    // create pool of fast-dfs client
    cli, err := fdfs.NewClient(cfg)
    if err != nil {
        return errors.Wrapf(err,
            "New Fast Dfs Client With %v", cfg)
    }
    // destory the pool
    defer cli.Destory()

    // upload
    fileID, err := cli.UploadByBuffer([]byte("foo-bar"), "test.txt")
    if err != nil {
        return errors.Wrap(err,
            "Upload to Fast Dfs")
    }

    // download
    _, err := cli.DownloadToBuffer(fileID, 0, 0)
    if err != nil {
        return errors.Wrap(err,
            "Download from Fast Dfs")
    }

    // delete
    err = cli.DeleteFile(fileID)
        if err != nil {
        return errors.Wrap(err,
            "Delete from Fast Dfs")
    }
}
```

## With Instance

``` go
// dfs.go
package dfs

import (
    "context"
    "runtime"
    "github.com/beiping96/grace"
    fdfsCli "github.com/beiping96/fdfs"
    "github.com/pkg/errors"
)

type Config = fdfsCli.Config

type Client interface {
    Get(remoteFileID string) (data []byte, err error)
    Set(ext string, data []byte) (remoteFileID string, err error)
    Del(remoteFileID string) (err error)

    Destory()
}

func NewClient(cfg *Config) (cli Client, err error) {
    sourceCli, err := fdfsCli.NewClient(cfg)
    if err != nil {
        return nil, errors.Wrapf(err,
            "github.com/beiping96/fdfs.NewClient(%+v)", cfg)
    }
    return &client{sourceCli}, nil
}

var _ Client = (*client)(nil)

type client struct{ *fdfsCli.Client }

func (cli *client) Get(remoteFileID string) (data []byte, err error) {
    return cli.Client.DownloadToBuffer(remoteFileID, 0, 0)
}

func (cli *client) Set(ext string, data []byte) (remoteFileID string, err error) {
    return cli.Client.UploadByBuffer(data, ext)
}

func (cli *client) Del(remoteFileID string) (err error) {
    return cli.Client.DeleteFile(remoteFileID)
}

var (
    constClient Client
)

func Construct(cfg Config) {
    var err error
    constClient, err = NewClient(&cfg)
    if err != nil {
        panic(errors.Wrapf(err, "New Dfs Client %+v", cfg))
    }
    grace.Go(clientDestoryMonitor)
}

func clientDestoryMonitor(ctx context.Context) {
    <-ctx.Done()
    runtime.Gosched()
    constClient.Destory()
}

func GetConstClient() Client { return constClient }
```

``` go
// main.go
package main

import (
    "context"
    "dfs"
    "github.com/beiping96/grace"
)

func main() {
    cfg := dfs.Config{
        Tracker: []string{"172.16.3.15:22122"},
        MaxConn: 100,
    }

    dfs.Construct(cfg)

    // async
    grace.Go(func(ctx context.Context) {
        fileID, _ :=dfs.GetConstClient().Set("test.txt", []byte("foo-bar"))
        dfs.GetConstClient().Del(fileID)
    })

    // running
    grace.Run(time.Second)
}
```

[![GoDoc](https://godoc.org/github.com/beiping96/fdfs?status.svg)](https://godoc.org/github.com/beiping96/fdfs)
