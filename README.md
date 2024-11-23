[![axone github banner](https://raw.githubusercontent.com/axone-protocol/.github/main/profile/static/axone-banner.png)](https://axone.xyz)

<p align="center">
  <a href="https://discord.gg/axone"><img src="https://img.shields.io/badge/Discord-7289DA?style=for-the-badge&logo=discord&logoColor=white" /></a> &nbsp;
  <a href="https://www.linkedin.com/company/axone-protocol/"><img src="https://img.shields.io/badge/LinkedIn-0077B5?style=for-the-badge&logo=linkedin&logoColor=white" /></a> &nbsp;
  <a href="https://twitter.com/axonexyz"><img src="https://img.shields.io/badge/Twitter-1DA1F2?style=for-the-badge&logo=twitter&logoColor=white" /></a> &nbsp;
  <a href="https://blog.axone.xyz"><img src="https://img.shields.io/badge/Medium-12100E?style=for-the-badge&logo=medium&logoColor=white" /></a> &nbsp;
  <a href="https://www.youtube.com/channel/UCiOfcTaUyv2Szv4OQIepIvg"><img src="https://img.shields.io/badge/YouTube-FF0000?style=for-the-badge&logo=youtube&logoColor=white" /></a>
</p>

# Axone Prolog Virtual Machine

[![build](https://img.shields.io/github/actions/workflow/status/axone-protocol/prolog/go.yml?label=build&style=for-the-badge&logo=github)](https://github.com/axone-protocol/prolog/actions/workflows/go.yml)
[![codecov](https://img.shields.io/codecov/c/github/axone-protocol/prolog?style=for-the-badge&token=O3FJO5QDCA&logo=codecov)](https://codecov.io/gh/axone-protocol/prolog)
[![GoDoc](https://img.shields.io/badge/godoc-reference-5272B4.svg?style=for-the-badge&logo=go)](https://pkg.go.dev/github.com/axone-protocol/prolog)
[![Go Report Card](https://goreportcard.com/badge/github.com/axone-protocol/prolog?style=for-the-badge)](https://goreportcard.com/report/github.com/axone-protocol/prolog)
[![conventional commits](https://img.shields.io/badge/Conventional%20Commits-1.0.0-yellow.svg?style=for-the-badge&logo=conventionalcommits)](https://conventionalcommits.org)
[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg?style=for-the-badge)](https://github.com/axone-protocol/.github/blob/main/CODE_OF_CONDUCT.md)
[![license](https://img.shields.io/github/license/axone-protocol/prolog.svg?label=License&style=for-the-badge)](https://opensource.org/license/mit)

> [!IMPORTANT]
> This repository is a _hard fork_ of [ichiban/prolog](https://github.com/ichiban/prolog), customized to meet the specific requirements of the [Axone protocol](https://github.com/axone-protocol). It is maintained independently for our use case, and upstream updates may not be regularly integrated.
>
> For the original, general-purpose Prolog implementation or to contribute to the broader community, please visit the [ichiban/prolog repository](https://github.com/ichiban/prolog).

## What is this?

`axone-protocol/prolog` is a Prolog virtual machine written in Go, designed to be embedded in blockchain environments.
It serves as the core of the [Axone protocol](https://axone.xyz) for decentralized, logic-based smart contracts.

This project is a fork of the [ichiban/prolog](https://github.com/ichiban/prolog) repository, striving to maintain
ISO standard compliance where feasible while adapting to the unique constraints of blockchain execution.

## Deviations from the ISO Standard

The following customizations have been made to adapt the original `ichiban/prolog` implementation to the blockchain environment:

- Capped variable allocation to limit the number of variables.
- Replaced maps with ordered maps to ensure deterministic execution.
- Implemented secure integer arithmetic for functors.
- Integrated [cockroachdb/apd](https://github.com/cockroachdb/apd) for floating-point arithmetic.
- Removed support for trigonometric functions (`sin`, `cos`, `tan`, `asin`, `acos`, `atan`).
- Introduced VM hooks for enhanced Prolog execution control.
- Added support for the `Dict` term.

## License

Distributed under the MIT license. See `LICENSE` for more information.

## Bug reports & feature requests

If you notice anything not behaving how you expected, if you would like to make a suggestion or would like
to request a new feature, please open a [**new issue**](https://github.com/axone-protocol/axoned/issues/new/choose). We appreciate any help
you're willing to give!

> Don't hesitate to ask if you are having trouble setting up your project repository, creating your first branch or
> configuring your development environment. Mentors and maintainers are here to help!

## You want to get involved? 😍

So you want to contribute? Great! ❤️ We appreciate any help you're willing to give. Don't hesitate to open issues and/or
submit pull requests.

We believe that collaboration is key to the success of the Axone project. Join our Community discussions on the [Community space](https://github.com/orgs/axone-protocol/discussions) to:

- Engage in conversations with peers and experts.
- Share your insights and experiences with Axone.
- Learn from others and expand your knowledge of the protocol.

The Community space serves as a hub for discussions, questions, and knowledge-sharing related to Axone.
We encourage you to actively participate and contribute to the growth of our community.

Please check out Axone health files:

- [Contributing](https://github.com/axone-protocol/.github/blob/main/CONTRIBUTING.md)
- [Code of conduct](https://github.com/axone-protocol/.github/blob/main/CODE_OF_CONDUCT.md)

## Acknowledgements

We would like to thank the following projects for their inspiration and for providing the foundation for this project:

- [ichiban](https://github.com/ichiban) for the original Prolog implementation.
