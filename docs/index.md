---
[Home](index.md) | [Getting Started](getting-started.md) | [Configuration](configuration.md) | [Usage](usage.md) | [Architecture](architecture.md) | [Docker Compose](docker-compose.md) | [Testing](testing.md) | [CI/CD & GitHub Actions](ci-cd.md) | [FAQ](faq.md) | [Contributing](contributing.md) | [Rules](rules.md)
---

# DOSync.io Documentation

## What You'll Find Here

- [Getting Started](getting-started.md): Quick installation and setup
- [Configuration](configuration.md): All options for registries, policies, and environment
- [Usage](usage.md): CLI, service, and advanced scenarios
- [Architecture](architecture.md): How DOSync works under the hood, including replica detection and update flow
- [Docker Compose Integration](docker-compose.md): Best practices and troubleshooting
- [Testing](testing.md): How to run and extend tests
- [CI/CD & GitHub Actions](ci-cd.md): Automate builds and deployments to registries
- [FAQ](faq.md): Answers to common questions
- [Contributing](contributing.md): How to get involved and our development workflow
- [Rules & Conventions](rules.md): Coding standards and links to rule files

For a quick start, see [Getting Started](getting-started.md).

## What is DOSync.io?

**[DOSync.io](https://dosync.io)** (pronounced "doo-sink-eeo") is an open-source automation tool—available at [DOSync.io](https://dosync.io)—designed to keep your Docker Compose services always running the latest container images from any registry. It was built to solve the pain of manual image updates, reduce deployment risk, and bring true zero-downtime updates to modern containerized environments.

The short name for the project is **DOSync**.

## Why was DOSync created?

Managing Docker Compose deployments at scale is challenging:

- **Manual image updates are error-prone and time-consuming.** Teams often forget to update images, miss critical security patches, or deploy inconsistent versions across environments.
- **Multi-registry support is complex.** Many organizations use images from Docker Hub, AWS ECR, DigitalOcean, GCR, GHCR, and private registries—each with different authentication and tag policies.
- **Zero-downtime is hard to achieve.** Rolling updates, blue-green deployments, and replica management require careful orchestration to avoid service interruptions.
- **Configuration is scattered.** Credentials, update policies, and backup strategies are often spread across scripts, environment variables, and CI/CD pipelines, making them hard to audit and maintain.

DOSync was created to address these challenges with a single, robust, and extensible solution.

## Main Problems DOSync Solves

- **Automates image updates** for all services in your Docker Compose stack, polling any supported registry for new tags and applying updates safely.
- **Supports all major registries** (Docker Hub, AWS ECR, DigitalOcean, GCR, GHCR, ACR, Quay, Harbor, and custom/private registries) with unified configuration.
- **Enables zero-downtime deployments** by detecting and updating service replicas intelligently, supporting both scale-based and name-based patterns (including blue-green and canary releases).
- **Centralizes configuration** in a single `dosync.yaml` file, making it easy to manage credentials, image policies, backup strategies, and API endpoints.
- **Provides observability and control** with built-in Metrics and Admin APIs for monitoring, health checks, and manual sync triggers.
- **Integrates seamlessly** with Docker Compose, systemd, and CI/CD pipelines, and can run as a container or standalone binary.

DOSync is ideal for teams and operators who want reliable, hands-off updates, improved security, and a clear audit trail for their container deployments.

---

## [Next ➡️ Getting Started](getting-started.md)
