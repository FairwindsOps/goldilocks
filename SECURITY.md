# Security Policy

Security is a top priority for the Goldilocks project. As a Kubernetes-native tool that manages Vertical Pod Autoscaler (VPA) objects and exposes a web dashboard within clusters, we take the security of our codebase and our users' clusters seriously.

We appreciate the efforts of the security research community in helping us keep our code and users safe.

## Supported Versions

We provide security updates for the following versions of Goldilocks:

| Version | Supported          |
| ------- | ------------------ |
| 5.x.x   | ✅ Yes              |
| 4.x.x   | ✅ Yes (Critical only) |
| < 4.0   | ❌ No               |

> **Note:** We strongly recommend running the latest release. Older minor releases within a supported major version will only receive patches for critical or high-severity vulnerabilities.

## Reporting a Vulnerability

If you discover a security vulnerability, please **do not** create a public GitHub issue.

### Preferred Reporting Method

Please report vulnerabilities through [GitHub Security Advisories](https://github.com/FairwindsOps/goldilocks/security/advisories/new) (preferred) or by emailing: **security@fairwinds.com**

### What to Include in Your Report

- A clear description of the vulnerability and its potential impact.
- Detailed steps to reproduce the issue (a proof-of-concept script, `kubectl` commands, or video is highly appreciated).
- The specific version(s) of Goldilocks affected.
- The Kubernetes version and VPA version used during testing.
- The deployment method used (Helm chart, raw manifests, etc.).
- Any potential mitigations you suggest.

### Our Response Time

| Stage                           | SLA             |
| ------------------------------- | --------------- |
| Acknowledgement of report       | **48 hours**    |
| Triage and validation           | **5 business days** |
| Patch development (Critical/High) | **14 business days** |
| Patch development (Medium/Low)  | **30 business days** |
| Coordinated public disclosure   | **90 days** (or upon fix release, whichever is sooner) |

We will keep you informed of our progress and coordinate a public disclosure date with you.

## Scope

### In Scope

The following components and attack surfaces are in scope for security reports:

- **Dashboard web server** — HTTP handler logic, routing, template rendering, and static file serving (`pkg/dashboard/`).
- **Kubernetes API interactions** — VPA creation, update, deletion, and RBAC authorization boundaries (`pkg/vpa/`, `pkg/kube/`).
- **Controller logic** — Namespace/Pod watch handlers and reconciliation logic (`pkg/controller/`, `pkg/handler/`).
- **Input validation** — Command-line flags, environment variable parsing, namespace parameters, and URL query parameters.
- **Template injection** — Server-side Go template rendering and any user-controllable data flowing into HTML output.
- **Client-side JavaScript** — XSS vectors, API token handling, and external fetch requests (`assets/js/`).
- **Container security** — Dockerfile hardening, base image vulnerabilities, and runtime user configuration.
- **Supply chain** — Build pipeline integrity, dependency integrity, and binary signing (`Makefile`, `.goreleaser.yml`).
- **Summary/CLI output** — File write operations (`--output-file`) and data serialization.

### Out of Scope

- **Denial of Service (DoS)** — Volumetric or resource exhaustion attacks against the dashboard or controller.
- **Social engineering** — Phishing or social engineering attacks against maintainers or contributors.
- **Third-party dependencies** — Vulnerabilities in upstream dependencies (e.g., `gorilla/mux`, `client-go`, VPA itself). Please report these to the respective upstream maintainers. However, reports about *how* Goldilocks uses a vulnerable dependency are welcome.
- **Kubernetes cluster misconfiguration** — Issues arising from overly permissive RBAC policies, network policies, or cluster-level misconfigurations not caused by Goldilocks.
- **Fairwinds Insights integration** — Security issues in the external `insights.fairwinds.com` API. Report those to Fairwinds directly.

## Security Best Practices for Operators

When deploying Goldilocks, we recommend:

1. **Restrict RBAC permissions** — Use the principle of least privilege for the Goldilocks service account. Only grant necessary permissions for VPA management.
2. **Network policies** — Restrict dashboard access to authorized users/networks. The dashboard does not include authentication; use a reverse proxy, service mesh, or Kubernetes network policies.
3. **Namespace isolation** — Use `--exclude-namespaces` and `--include-namespaces` flags to limit the controller's scope.
4. **Keep dependencies updated** — Regularly update Goldilocks and its dependencies. Enable Dependabot alerts on your fork.
5. **Verify release signatures** — All releases are signed with [cosign](https://github.com/sigstore/cosign). Verify signatures before deployment:
   ```bash
   cosign verify us-docker.pkg.dev/fairwinds-ops/oss/goldilocks:<tag> \
     --key https://artifacts.fairwinds.com/cosign-p256.pub
   ```

## Acknowledgments

We gratefully acknowledge the security researchers who responsibly disclose vulnerabilities and help us improve the security posture of Goldilocks. Contributors will be credited in release notes (unless they prefer to remain anonymous).
