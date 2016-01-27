# go double buffering

A simple library to manage resources which can be upgraded gracefully. This blog shows how to use it: http://blog.codeg.cn/2016/01/27/double-buffering/


During the continuous running period of a network server, there are some resources need to be updated on real-time. How to do an elegant resource upgrading? Here we propose a method to solve this problem which is called "double buffering" technology.

When double buffering is enabled, all resource upgrading operations are first initialized to a buffer handler instead of the handler which is being used by the process. After all initializing operations are completed, the buffer handler is swapped directly to the old one associated with it.
