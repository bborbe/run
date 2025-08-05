# Changelog

All notable changes to this project will be documented in this file.

Please choose versions by [Semantic Versioning](http://semver.org/).

* MAJOR version when you make incompatible API changes,
* MINOR version when you add functionality in a backwards-compatible manner, and
* PATCH version when you make backwards-compatible bug fixes.

## v1.7.6

- Add comprehensive Go documentation following best practices to all public APIs
- Create package documentation (doc.go) with usage examples 
- Update README with detailed library documentation and examples
- Improve function comments for better godoc rendering
- Update generated mocks (HasCaptureException interface)

## v1.7.5

- go mod update
- update mocks

## v1.7.4

- add tests
- go mod update

## v1.7.3

- add tests

## v1.7.2

- refactor
- add tests
- go mod update

## v1.7.1

- MultiTrigger.Add returns Trigger instead Fire

## v1.7.0

- add ContextWithSig
- go mod update

## v1.6.0

- remove vendor
- go mod update

## v1.5.7

- go mod update

## v1.5.6

- go mod update

## v1.5.5

- go mod update

## v1.5.4

- go mod update

## v1.5.3

- go mod update
- replace pkg/errors

## v1.5.2

- go mod update

## v1.5.1

- return Func

## v1.5.0

- add backoff factor 

## v1.4.0

- add background runner

## v1.3.1

- use github.com/bborbe/errors for better error list display

## v1.3.0

- retry check if err is retryable
- update deps and add vulncheck

## v1.2.0

- use errors join

## v1.1.0

- use ginkgo v2
- improve use of counterfeiter

## v1.0.0

- Initial Version
