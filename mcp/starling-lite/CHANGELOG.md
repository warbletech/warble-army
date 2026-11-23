# Changelog — Starling Lite

Managed by release-please (component `starling-lite`, tags `starling-lite-vX.Y.Z`).
See [docs/RELEASING.md](../../docs/RELEASING.md).

## [0.4.0](https://github.com/warbletech/warble-army/compare/starling-lite-v0.3.0...starling-lite-v0.4.0) (2026-11-21)


### Features

* heron-ops submodule, starling-ex-ce Helm chart, warbleoss.org links ([4c256f4](https://github.com/warbletech/warble-army/commit/4c256f4dfda0182a0be1986a627af7abd8e30c9e))
* Railway/Fly always-on Starling Lite deploy (PORT-aware) ([2698fb1](https://github.com/warbletech/warble-army/commit/2698fb1b0984ae7d72566bc3d3b5665954507ecc))
* **sandbox:** per-project sandbox namespace chart + sandbox_project MCP tool ([#21](https://github.com/warbletech/warble-army/issues/21)) ([10df432](https://github.com/warbletech/warble-army/commit/10df4324a89a13c69c043e801b31913e38f4bff3)), closes [#20](https://github.com/warbletech/warble-army/issues/20)
* **starling-lite:** guarded LLM fallback propose_remediation_llm ([#28](https://github.com/warbletech/warble-army/issues/28)) ([ffb7461](https://github.com/warbletech/warble-army/commit/ffb746167e146373fd0d59362d94b35c28e0db3d))
* **starling-lite:** MCP 2026-07-28 preferred (stateless / MCP 2.0 era) ([1db32ec](https://github.com/warbletech/warble-army/commit/1db32ec3e603605351769a3a5f3c3c25f08b2afe))
* **starling-lite:** MCP facade + reflexion agent scaffold ([2ef40b9](https://github.com/warbletech/warble-army/commit/2ef40b9d79193a386b7436e1e71bde20e8486197))


### Bug Fixes

* **deps:** bump golang.org/x/crypto, x/net, x/text for known vulns ([7e0de20](https://github.com/warbletech/warble-army/commit/7e0de204865c391a77015ec624d2c7693aaf0ee7))
* **starling-lite:** bump k8s.io modules together and use Go 1.26 ([#35](https://github.com/warbletech/warble-army/issues/35)) ([#37](https://github.com/warbletech/warble-army/issues/37)) ([8947320](https://github.com/warbletech/warble-army/commit/8947320c3b5e387df16fab1389e59d0c5c1485c1))
* **starling-lite:** flag incomplete scans, real container names in remediation patches, ship scanners in images ([f6c5367](https://github.com/warbletech/warble-army/commit/f6c53675f4601abc4c4f7940ccddfa1f8c97ea59))
* **starling-lite:** mark incomplete scans, real container names in remediation patches ([99217cb](https://github.com/warbletech/warble-army/commit/99217cb5714703fc495e49d8b200c994ab48ed8f))
* **starling-lite:** scan-only demo without cluster errors; read-only demo-cluster identity ([#46](https://github.com/warbletech/warble-army/issues/46)) ([ce76c7f](https://github.com/warbletech/warble-army/commit/ce76c7fc356bf286865ed4931bab7c53a328aae5))

## [Unreleased]
