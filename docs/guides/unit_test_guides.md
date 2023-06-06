# Unit Tests

To enhance code quality and maintainability, unit tests play a crucial role in verifying the functionality of isolated functions. We highly encourage and appreciate contributions of unit tests to EdgeNet. Please feel free to contribute by adding unit tests to the codebase. You can find some of the references and guides in this document.

## What are Unit Tests?
Unit testing is an essential part of software development in Go. It helps ensure the reliability and correctness of individual units of code. This document provides guidelines for writing effective unit tests in Go to enhance the quality and maintainability of your project.

## Guides to Follow

1. Test File Organization

    Place your unit tests in a separate directory named `*_test.go`, alongside the package they are testing. For example, if your package is named `foo`, your test files should be named `foo_test.go`. Follow the same directory structure as the main package to keep tests and source code synchronized.

2. Test Function Naming and Function Signature

    Begin the name of your test functions with Test followed by a descriptive name that highlights the functionality being tested. For example, `TestAddition()` or `TestCalculateTotal()`. Use camel case for the descriptive part of the test function names.

    The test functions should have the signature `func TestFoo(t *testing.T)`, where `Foo` is the name of the functionality being tested. Use the `*testing.T` type parameter to report failures and manage the test state.

3. Table-Driven Testing

    Utilize table-driven testing to test multiple scenarios with different inputs and expected outputs. Define a slice of test cases and iterate over them, calling the test function for each case. This approach improves test coverage and makes it easier to add, modify, or remove test cases. Additionally, to encapsulate common testing logic in helper functions to avoid code duplication. Test helper functions can be used to set up test data, perform assertions, or handle complex test scenarios. Keep the helper functions in the same test file or a separate package if they are used across multiple tests.

    ```go

    func TestAddition(t *testing.T) {
        cases := []struct {
            a, b     int
            expected int
        }{
            {2, 3, 5},
            {-1, 1, 0},
        }

        for _, tc := range cases {
            result := Add(tc.a, tc.b)
            if result != tc.expected {
                t.Errorf("Add(%d, %d) = %d, expected %d", tc.a, tc.b, result, tc.expected)
            }
        }
    }
    ```

5. Using Interfaces and Avoiding I/O

    In Go, the functions are defined in interfaces and then implemented separately. This idea comes in handy when writing tests. For example, in a function that should read a file, you can use `io.Reader` interface. This interface defines some functions for reading from a byte sequence. In the real world, you can open a file and in the test case, you can use `strings.NewReader(s string)` to create an `io.Reader` interface that reads the given string.

6. Kubernetes Fake Clientset and Other Libraries

    When it comes to testing custom controllers, the ["k8s.io/client-go/kubernetes/fake"](https://pkg.go.dev/k8s.io/client-go/kubernetes/fake) package proves to be a valuable tool. By utilizing this package, a fake clientset can be generated, enabling the execution of Kubernetes API calls for testing purposes. An excellent illustration of this approach can be observed in the sample-controller test file of the Kubernetes example project, which can be accessed [here](https://github.com/kubernetes/sample-controller/blob/master/controller_test.go).

    Other testing libraries provide different functionalities. These are given below.

      - [AFERO](https://github.com/spf13/afero): A filesystem abstraction for go.

      - [ftest](https://godocs.io/testing/fstest): An inmemory filesystem for testing.

      - [ioutil.TempFile()](https://pkg.go.dev/io/ioutil#TempFile) and [ioutil.TempDir()](https://pkg.go.dev/io/ioutil#TempDir): Implementation for creating temp files and directories.

      - [httptest](https://pkg.go.dev/net/http/httptest): A testing https server implementation that imitates real server communication.

      - [sshtest](https://github.com/folbricht/sshtest): A testing library that implements an ssh server communication.

7. Test Documentation

    Keep your test cases and assertions descriptive and clear to understand their purpose and expected behavior. Use comments to provide additional context or explanations where necessary. Document any test assumptions, prerequisites, or limitations to guide developers using the tests.

## Conclusion

Following these guidelines will help you write well-structured and maintainable unit tests for your Go project. Effective unit testing improves the reliability of your codebase, detects bugs early, and supports long-term project maintenance. Remember that unit tests should cover various scenarios and edge cases to achieve comprehensive coverage. Regularly review and update your tests as your project evolves to ensure their continued effectiveness.
