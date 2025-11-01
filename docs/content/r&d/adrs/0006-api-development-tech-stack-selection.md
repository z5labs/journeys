---
title: "[0006] API Development Tech Stack Selection"
description: >
    Selection of the technology stack for building the REST API backend, including programming language, framework, and core libraries.
type: docs
weight: 6
status: "proposed"
date: 2025-11-01
deciders: []
consulted: []
informed: []
---

## Context and Problem Statement

We need to select a Go framework and supporting libraries for developing the REST API backend for our journey tracking system. As Go is the default language for all Z5Labs projects, this decision focuses on choosing the right framework and architectural approach. The chosen framework should support rapid development, provide good observability capabilities, integrate well with our authentication/authorization decisions (OAuth2/OIDC, OPA/OpenFGA), and align with modern REST API best practices.

<!-- This is an optional element. Feel free to remove. -->
## Decision Drivers

* Developer productivity and ease of defining REST endpoints
* Built-in or easy integration with OpenTelemetry for observability
* Support for structured logging and configuration management
* Simplicity and minimal boilerplate code
* Request/response serialization and validation
* Middleware support for cross-cutting concerns (auth, logging, metrics)
* Integration with OAuth2/OIDC and authorization systems (OPA/OpenFGA)
* Compatibility with Z5Labs development practices and existing projects
* Active maintenance and community support

## Considered Options

* Go with z5labs/humus framework
* Go with stdlib net/http + chi router
* Go with Gin framework
* Go with Echo framework
* Go with Fiber framework

## Decision Outcome

Chosen option: "Go with z5labs/humus framework", because it provides the best alignment with Z5Labs development practices and includes built-in support for essential cross-cutting concerns (OpenTelemetry observability, configuration management, structured logging) that would otherwise require significant integration effort. While it has a smaller community compared to popular frameworks like Gin or Echo, the benefits of internal maintainability, consistent patterns across Z5Labs projects, and reduced boilerplate code outweigh the drawbacks.

### Consequences

* Good, because all Z5Labs projects will share consistent patterns for REST API development
* Good, because OpenTelemetry instrumentation is available out-of-the-box without additional integration work
* Good, because configuration management through embedded YAML files simplifies deployment
* Good, because the RPC-style operation pattern (`rpc.NewOperation`, `rpc.ConsumeJson`, `rpc.ReturnJson`) provides clean separation of concerns
* Good, because structured logging with `slog` is built-in and standardized
* Good, because internal support and expertise is readily available within Z5Labs
* Bad, because onboarding new developers may require learning framework-specific patterns
* Bad, because the smaller ecosystem means fewer third-party examples and community resources
* Bad, because any framework limitations or bugs require internal fixes rather than relying on large community

### Confirmation

This decision will be confirmed by:
* Implementation of REST API using the humus framework in the `api/` directory
* All endpoint handlers following the humus RPC-style operation pattern
* Consistent use of `rest.Run()`, `rest.Api`, and RPC operations throughout the codebase
* OpenTelemetry integration via humus framework utilities
* Configuration management using embedded YAML files

<!-- This is an optional element. Feel free to remove. -->
## Pros and Cons of the Options

### Go with z5labs/humus framework

https://github.com/z5labs/humus

A Z5Labs-maintained framework providing REST API scaffolding, RPC-style operations, embedded configuration, and built-in OpenTelemetry support.

* Good, because it's developed and maintained by Z5Labs, ensuring alignment with internal practices
* Good, because it has built-in OpenTelemetry integration for metrics, traces, and logs
* Good, because it provides embedded configuration management (YAML)
* Good, because it offers RPC-style operation handlers with clean JSON serialization patterns
* Good, because it uses standard library `slog` for structured logging
* Good, because it reduces boilerplate with `rest.Run()` and `rest.Api` abstractions
* Neutral, because it's a less mature ecosystem compared to popular frameworks like Gin or Echo
* Neutral, because community support and third-party examples are limited
* Bad, because documentation may be limited compared to widely-adopted frameworks
* Bad, because hiring developers familiar with the framework may be challenging

### Go with stdlib net/http + chi router

https://github.com/go-chi/chi

Using Go's standard library HTTP server with the chi lightweight router for REST routing.

* Good, because it relies primarily on standard library, ensuring long-term stability
* Good, because chi is a minimal, idiomatic router that composes well with stdlib
* Good, because it provides full control over middleware and request handling
* Good, because it has a large community and extensive examples
* Neutral, because it requires manually integrating OpenTelemetry, logging, and configuration
* Neutral, because it provides maximum flexibility but requires more setup code
* Bad, because it requires more boilerplate for common patterns (JSON serialization, validation)
* Bad, because developers must build or integrate observability patterns themselves

### Go with Gin framework

https://github.com/gin-gonic/gin

A popular high-performance HTTP web framework with a martini-like API.

* Good, because it's one of the most popular Go web frameworks with extensive community support
* Good, because it has excellent performance and low memory footprint
* Good, because it provides built-in JSON validation and serialization
* Good, because it has rich middleware ecosystem
* Good, because extensive documentation and examples are available
* Neutral, because it uses its own context type rather than standard `context.Context`
* Bad, because OpenTelemetry integration requires additional setup
* Bad, because it doesn't provide configuration management out of the box
* Bad, because it uses a less idiomatic API style compared to stdlib patterns

### Go with Echo framework

https://echo.labstack.com/

A high-performance, extensible, minimalist Go web framework.

* Good, because it's well-documented with good performance characteristics
* Good, because it provides built-in middleware for common tasks
* Good, because it has automatic JSON binding and validation
* Good, because it has a clean, intuitive API
* Good, because it supports standard `context.Context`
* Neutral, because it has good but not exceptional community size
* Bad, because OpenTelemetry integration requires additional setup
* Bad, because configuration management must be built separately
* Bad, because it's not aligned with Z5Labs existing practices

### Go with Fiber framework

https://gofiber.io/

An Express-inspired web framework built on top of Fasthttp.

* Good, because it has excellent performance (uses fasthttp instead of net/http)
* Good, because it has an intuitive Express-like API familiar to Node.js developers
* Good, because it has extensive built-in middleware
* Good, because it has good documentation and examples
* Neutral, because it uses fasthttp instead of standard library net/http
* Bad, because it doesn't use standard library patterns, creating potential compatibility issues
* Bad, because OpenTelemetry integration is less straightforward
* Bad, because using fasthttp may limit compatibility with standard Go HTTP tooling
* Bad, because it's not aligned with Z5Labs existing practices

## More Information

### Related Decisions
* ADR-0002: SSO Authentication Strategy - The humus framework will integrate with OAuth2/OIDC providers
* ADR-0003: OAuth2/OIDC Provider Selection - Humus endpoints will handle callbacks from Google, Facebook, and Apple
* ADR-0004: Session Management - JWT token handling will be implemented using humus RPC operations

### Key Framework Features Used
* `rest.Run()` - Application bootstrapping with embedded configuration
* `rest.Api` - REST API instance for registering routes
* `rpc.NewOperation()` - RPC-style operation wrapper for handlers
* `rpc.ConsumeJson()` - Automatic JSON request deserialization
* `rpc.ReturnJson()` - Automatic JSON response serialization
* `humus.Logger()` - Structured logging with slog

### Framework Repository
* GitHub: https://github.com/z5labs/humus
* Maintained by: Z5Labs team
