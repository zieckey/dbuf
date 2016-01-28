# go double buffering

A simple library to manage resources which can be upgraded gracefully.
This blog written in Chinese the usage of this library : http://blog.codeg.cn/2016/01/27/double-buffering/


During the continuous running period of a network server, there are some resources need to be updated on real-time.
How to do an elegant resource upgrading? Here we propose a method to solve this problem which is called "double buffering" technology.

When double buffering is enabled, all resource upgrading operations are first initialized to a buffer handler instead of the handler
which is being used by the process. After all initializing operations are completed, the buffer handler is swapped directly to the old one associated with it.

# Example

There is a simple example located in the `example` directory.
The example provides a simple querying service which require two parameters: id and query.
`id` is used as an identity of the client. `query` is used as the entity the querying. 
The service logic uses the `query` as a key to query a backend database and return the result.
Avoiding vicious client to do request we provide an black-id-list which holds all the black ids.

Using the following steps to run this example:

### Step 1

```shell
$ git clone https://github.com/zieckey/go-doublebuffering
$ cd go-doublebuffering/example
$ go build
$ ./example ./black_id.txt
```

### Step 2

And in another console, we can do some requests:

```shell
$ curl "http://localhost:8091/q?id=123&query=jane"
hello, 123
$ curl "http://localhost:8091/q?id=475e5a499587a52ea14a23031ecce7c9&query=jane"
ERROR
```

`id=123` is OK. But `id=475e5a499587a52ea14a23031ecce7c9` is rejected.

### Step 3

Then we added `123` to the black-id-list.
Now we have updated the black-id-list file and need it to be loaded into the service process,
what we need to do is only to send a administration request to have it enabled. Like this:

```shell
$ curl "http://localhost:8091/admin/reload?name=black_id&path=./black_id.txt"
OK
$ curl "http://localhost:8091/q?id=123&query=jane"
ERROR
```

`id=123` is rejected now. It works.