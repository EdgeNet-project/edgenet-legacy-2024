# Unit Tests

There are some tips to increase code quality and maintainability:

- Apply clean code philosophy for example implement functions short and give them a single task.
- Write functions as agnostic as possible. Use abstractions, interfaces, and mocks.
- Measure the performance of the code through benchmarks and load tests.
- Understand business logic better by testing regular and edge cases.

## Table-Driven Testing
This is one of the simplest ways of testing in go. Using a standard testing library a set of inputs and outputs are created. The created values can also be randomly generated.

## Test Suites 
Sometimes you need to initialize a variable and run a setup. For these cases, Test Suites are used. Here you need to implement the `suite.Suite` interface. You can also combine this with table-driven testing.

## Using Interfaces and Avoiding I/O
In Go, the functions are defined in interfaces and then implemented separately. This is a very important idea in programming.

For example, in a function that should read a file, you can use `io.Reader` interface. This interface defines some functions for reading from a byte sequence. In the real world, you can open a file and in the test case, you can use `strings.NewReader(s string)` to create an `io.Reader` interface that reads the given string.

If you need to create a file and interact with it differently, then you can use these libraries.
- [AFERO](https://github.com/spf13/afero): A filesystem abstraction for go.

- [ftest](https://godocs.io/testing/fstest): An inmemory filesystem for testing.

- [ioutil.TempFile()](https://pkg.go.dev/io/ioutil#TempFile) and [ioutil.TempDir()](https://pkg.go.dev/io/ioutil#TempDir): Implementation for creating temp files and directories.

There is also a library for testing http requests.

- [httptest](https://pkg.go.dev/net/http/httptest): A testing https server implementation that imitates real server communication.

- [sshtest](https://github.com/folbricht/sshtest): A testing library that implements an ssh server communication.

## Testing Custom Controllers
While writing unit tests for custom controllers please apply the mentioned practices. Additionally, Kubernetes example [sample-controller](https://github.com/kubernetes/sample-controller) provides a good basis for testing controller code. 

## Benchmarking
While testing the code it is also important to see how fast the code executes. For this go provides benchmarking inside the testing package. The following code contains an example for benchmarking:

## References
- [1.](https://reshefsharvit.medium.com/5-tips-for-better-unit-testing-in-golang-b25f9e79885a) Unit Testing Practices and Tips.
- [2.](https://fossa.com/blog/golang-best-practices-testing-go) Best Practices Testing Go. 
- [3.](https://stackoverflow.com/questions/49418982/unit-testing-an-ssh-client-in-go) SSH unit testing library for go.