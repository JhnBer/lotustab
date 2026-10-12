# Contributing to LotusTab

Thanks for contributing. This guide covers how to send changes and how licensing works. The full Contributor License Agreement is in [CLA.md](CLA.md) — do not copy that legal text here.

## How to contribute

1. Fork the repository and create a branch from `main`.
2. Make focused changes. Follow [AGENTS.md](AGENTS.md) for architecture (drivers as adapters, thin Wails bindings, secrets in the keychain, shadcn-vue + Tailwind only).
3. Open a pull request against `main`.
4. On your first PR, **sign the CLA** when [CLA Assistant](https://cla-assistant.io/) asks you to. Unsigned human PRs cannot merge while the CLA check is required.
5. Automated bots (for example Dependabot) are allowlisted and do not need to sign.

Contributor history lives in GitHub / git (`git shortlog`, the Contributors graph). We do **not** maintain a separate in-repo list of CLA signatures.

## Licenses

| Direction | License |
|---|---|
| **Inbound** (what you grant when a PR is accepted) | **MIT** |
| **Outbound** (how LotusTab is published) | **[GPLv3](LICENSE)** |

An accepted pull request means your Contribution is licensed to the Project under the MIT terms described in [CLA.md](CLA.md). The distributed product remains GPLv3 unless maintainers say otherwise for a specific release.

## Intent vs reserved rights

CLA and contribution policy exist to reduce legal risk as the product evolves — not to systematically move community features out of the free open build into a paid tier.

Functionality accepted into the open version stays available in open by default. We do not plan to make it premium-only only because it came from a community contribution. A contribution may still be used (including together with other code) in premium or cloud features.

Maintainers may, for architectural or business reasons, remove functionality from the public open version while keeping it in a premium build or related closed service, within the rights granted by the CLA / inbound MIT license.

## Development pointers

- Local setup and packaging: see [README.md](README.md).
- Prefer small PRs with a clear description of what changed and how you checked it.
- Do not commit secrets, credentials, or customer data.
