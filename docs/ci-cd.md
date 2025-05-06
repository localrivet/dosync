---
[Home](index.md) | [Getting Started](getting-started.md) | [Configuration](configuration.md) | [Usage](usage.md) | [Architecture](architecture.md) | [Docker Compose](docker-compose.md) | [Testing](testing.md) | [CI/CD & GitHub Actions](ci-cd.md) | [FAQ](faq.md) | [Contributing](contributing.md) | [Rules](rules.md)
---

# CI/CD & GitHub Actions

Automate your container builds and deployments to registries using GitHub Actions. This guide provides ready-to-use workflow examples and best practices for integrating DOSync into your CI/CD pipeline.

## Why Use CI/CD for Containers?

- **Consistency:** Ensure every image is built and tagged the same way, every time.
- **Security:** Automate patching and updates, reducing manual errors.
- **Speed:** Deploy new versions to registries and production faster.
- **Auditability:** Every build and deployment is tracked in your repository.

## Example: Build & Push to Docker Hub

This workflow builds your Docker image and pushes it to Docker Hub on every push to `main`:

```yaml
name: Build and Push Docker Image (Docker Hub)
on:
  push:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Log in to Docker Hub
        uses: docker/login-action@v3
        with:
          username: ${{ secrets.DOCKERHUB_USERNAME }}
          password: ${{ secrets.DOCKERHUB_TOKEN }}

      - name: Build and push
        uses: docker/build-push-action@v5
        with:
          context: .
          push: true
          tags: yourdockerhubuser/yourimage:latest
          # Optionally add more tags, e.g. ${{ github.sha }}

      - name: Upload Compose and dosync.yaml (optional)
        uses: actions/upload-artifact@v4
        with:
          name: compose-and-config
          path: |
            docker-compose.yml
            dosync.yaml
```

> **Tip:** Store your Docker Hub credentials as [GitHub Secrets](https://docs.github.com/en/actions/security-guides/encrypted-secrets).

## Example: Build & Push to GitHub Container Registry (GHCR)

```yaml
name: Build and Push Docker Image (GHCR)
on:
  push:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Log in to GHCR
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Build and push
        uses: docker/build-push-action@v5
        with:
          context: .
          push: true
          tags: ghcr.io/${{ github.repository_owner }}/yourimage:latest
```

> **Tip:** For private repositories, you may need to [create a Personal Access Token](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry#authenticating-to-the-container-registry) with `write:packages` scope and use it as a secret.

## Adapting for Other Registries

- **AWS ECR:** Use [aws-actions/amazon-ecr-login](https://github.com/aws-actions/amazon-ecr-login) and update the `login` and `tags` fields.
- **Google GCR/Artifact Registry:** Use [google-github-actions/auth](https://github.com/google-github-actions/auth) and [google-github-actions/setup-gcloud](https://github.com/google-github-actions/setup-gcloud).
- **Azure ACR:** Use [azure/docker-login](https://github.com/Azure/docker-login) and update the registry URL.
- **Quay.io, Harbor, DigitalOcean:** Use the `docker/login-action` with the appropriate registry URL and credentials.

See [Configuration](configuration.md) for details on setting up registry credentials and image policies.

## Best Practices

- **Use secrets for all credentials.** Never hard-code passwords or tokens in your workflow files.
- **Tag images with both `latest` and a unique value** (e.g., `${{ github.sha }}` or a version number) for traceability.
- **Upload your Compose and `dosync.yaml` files as artifacts** if you want to deploy them elsewhere or trigger DOSync updates in another environment.
- **Automate DOSync runs** by triggering a deployment or sync job after pushing new images.
- **Keep your Compose and `dosync.yaml` files in sync** with your image tags and registry configuration.

---

[⬆️ Back to Home](index.md)

---
