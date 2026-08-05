# Changelog

## 1.0.0 (2026-08-05)


### ⚠ BREAKING CHANGES

* **deps:** Update module gitlab.com/gitlab-org/api/client-go (v1.46.0 → v2.39.0)

### Features

* default talosconfig path and use GitHub App client-id auth ([#6](https://github.com/home-operations/downflate/issues/6)) ([ce72b1e](https://github.com/home-operations/downflate/commit/ce72b1ed10d524bf482d3a1fb1f9410191a57ff4))
* **deps:** Update module gitlab.com/gitlab-org/api/client-go (v1.46.0 → v2.39.0) ([dcaf58c](https://github.com/home-operations/downflate/commit/dcaf58c0f7ae3571c04d677548cdc8610ef50802))
* **deps:** update module gitlab.com/gitlab-org/api/client-go/v2 (v2.39.0 → v2.42.0) ([#9](https://github.com/home-operations/downflate/issues/9)) ([83a5d44](https://github.com/home-operations/downflate/commit/83a5d447a6c9399ac6a72e3988ae257d79c44cfc))
* **deps:** update module gitlab.com/gitlab-org/api/client-go/v2 (v2.42.0 → v2.43.0) ([#13](https://github.com/home-operations/downflate/issues/13)) ([aa459fa](https://github.com/home-operations/downflate/commit/aa459fac9919d3524c6a6ecfb13f64d020dc6b96))
* **deps:** update module gitlab.com/gitlab-org/api/client-go/v2 (v2.43.0 → v2.44.0) ([#15](https://github.com/home-operations/downflate/issues/15)) ([41ff89b](https://github.com/home-operations/downflate/commit/41ff89b5fd0288d799cff11f1149f84aef9501a2))


### Bug Fixes

* branches ([754a613](https://github.com/home-operations/downflate/commit/754a6139a35f0def9db90a62f7b7a13c019c4e53))
* **ci:** fail the merge gate on cancelled jobs, and key the lint cache on the toolchain ([#37](https://github.com/home-operations/downflate/issues/37)) ([437752a](https://github.com/home-operations/downflate/commit/437752a54b0ecd020287ef31c9b15f17bd47f491))
* **deps:** update module github.com/home-operations/flate (v0.4.8 → v0.4.9) ([#5](https://github.com/home-operations/downflate/issues/5)) ([038cd93](https://github.com/home-operations/downflate/commit/038cd93e43e754502a3362a97eb7869a9a09f023))
* **deps:** update module github.com/home-operations/flate (v0.4.9 → v0.4.10) ([#14](https://github.com/home-operations/downflate/issues/14)) ([3071eea](https://github.com/home-operations/downflate/commit/3071eeaa67daab247d2576c44c55ca4027868433))
* **deps:** update module github.com/siderolabs/talos/pkg/machinery (v1.13.4 → v1.13.5) ([#11](https://github.com/home-operations/downflate/issues/11)) ([1e6fbbc](https://github.com/home-operations/downflate/commit/1e6fbbc81f3c88c9c060381a5da42f0a7c17d283))
* **go:** update module github.com/home-operations/flate (v0.4.10 → v0.4.12) ([#23](https://github.com/home-operations/downflate/issues/23)) ([89c0686](https://github.com/home-operations/downflate/commit/89c068632a9248e8e1fa7a803d5436af05145091))
* nodes -&gt; node ([b6c7e0b](https://github.com/home-operations/downflate/commit/b6c7e0bc11945ea0598eab601a93bd272fec871d))


### Documentation

* add AGENTS.md with Go conventions ([#40](https://github.com/home-operations/downflate/issues/40)) ([d485ff9](https://github.com/home-operations/downflate/commit/d485ff9e095586d9f00b75b38dbdf0256b6884c4))


### Styles

* indent markdown at 2 to match embedded yaml ([#24](https://github.com/home-operations/downflate/issues/24)) ([90c97f9](https://github.com/home-operations/downflate/commit/90c97f91b6def2f829b91f4fd296662841a3d28b))


### Build System

* **mise:** add actionlint and refresh the lockfile ([#29](https://github.com/home-operations/downflate/issues/29)) ([0ec1a4b](https://github.com/home-operations/downflate/commit/0ec1a4beb9d5673b00c4293b996546f30f1cb8e1))


### Continuous Integration

* gate pull requests on a single Build Success check ([#28](https://github.com/home-operations/downflate/issues/28)) ([956693c](https://github.com/home-operations/downflate/commit/956693c0cf8bb4e95450c456eb927e5e8aa9fcde))
* **github-action:** Update action actions/cache (v5.0.5 → v6.0.0) ([#12](https://github.com/home-operations/downflate/issues/12)) ([d162bd8](https://github.com/home-operations/downflate/commit/d162bd8266998e24c7fcaf368181b7a3d5d2dd03))
* **github-action:** Update action actions/cache (v6.0.0 → v6.1.0) ([76ec1a0](https://github.com/home-operations/downflate/commit/76ec1a0eaf8983146638971cecfd4e6524a0ac9c))
* **github-action:** Update action actions/checkout (v6.0.3 → v7.0.0) ([#7](https://github.com/home-operations/downflate/issues/7)) ([f07324e](https://github.com/home-operations/downflate/commit/f07324e76aa74267785b79746daa4de5d36dfbd0))
* **github-action:** Update action actions/checkout (v7.0.0 → v7.0.1) ([f7e8942](https://github.com/home-operations/downflate/commit/f7e894236f0dc1f53cccd6c89165dfa11990e546))
* **github-action:** Update action actions/stale (v10.3.0 → v10.4.0) ([82ad000](https://github.com/home-operations/downflate/commit/82ad00050387505db3fe1c656262fc0d52a0483b))
* **github-action:** Update action actions/stale (v10.4.0 → v11.0.0) ([#38](https://github.com/home-operations/downflate/issues/38)) ([116dc78](https://github.com/home-operations/downflate/commit/116dc785661b45ee0edd3ba9b4650d95593a2108))
* **github-action:** Update action docker/github-builder (v1.12.0 → v1.13.0) ([fe89c85](https://github.com/home-operations/downflate/commit/fe89c855f07dd874f317a0cff8849402a92b23a4))
* **github-action:** Update action docker/github-builder (v1.13.0 → v1.14.0) ([e94f77e](https://github.com/home-operations/downflate/commit/e94f77ea3a98b0b0c3a1cbb1069d81a1e1955e18))
* **github-action:** Update action docker/github-builder (v1.14.0 → v1.15.0) ([#36](https://github.com/home-operations/downflate/issues/36)) ([5553978](https://github.com/home-operations/downflate/commit/5553978c0fc1c342320c4e830c4650347250d7ed))
* **github-action:** Update action home-operations/.github/actions/workflow-lint (v1.0.2 → v1.0.3) ([#45](https://github.com/home-operations/downflate/issues/45)) ([6b40ecb](https://github.com/home-operations/downflate/commit/6b40ecb1447c314fe98f3bc9da3e1ce2c39ec92b))
* **github-action:** Update action jdx/mise-action (v4.1.0 → v4.2.0) ([2ed7ed2](https://github.com/home-operations/downflate/commit/2ed7ed28b5c213f099c9957eea87a4c52e6bb0e3))
* **github-action:** Update action jdx/mise-action (v4.2.0 → v4.2.1) ([787373f](https://github.com/home-operations/downflate/commit/787373fa6c88c046b24b3a6a4a1bb1d862502746))
* **github-action:** Update action jdx/mise-action (v4.2.1 → v4.2.2) ([#31](https://github.com/home-operations/downflate/issues/31)) ([e2c7125](https://github.com/home-operations/downflate/commit/e2c71255769fa4459e45d906e7dc36c9ee464761))
* **github-action:** Update action jdx/mise-action (v4.2.2 → v4.2.3) ([#33](https://github.com/home-operations/downflate/issues/33)) ([9da7eba](https://github.com/home-operations/downflate/commit/9da7eba34f42eef583dbba88515c9854e79b2519))
* **github-action:** Update action jdx/mise-action (v4.2.3 → v4.2.4) ([#46](https://github.com/home-operations/downflate/issues/46)) ([b5df4b6](https://github.com/home-operations/downflate/commit/b5df4b69a91841ee9dbf5b6b2508d7b8300f6261))
* **github-action:** update workflow-lint action (1.0.0 → v1.0.2) ([#42](https://github.com/home-operations/downflate/issues/42)) ([97d5d4f](https://github.com/home-operations/downflate/commit/97d5d4fe7b253e09f3ae4e50ced1e891a7a1dd71))
* lint workflows with the shared composite action ([#30](https://github.com/home-operations/downflate/issues/30)) ([4678944](https://github.com/home-operations/downflate/commit/467894491145c0f39c3b4288c3cfa185da8f0d7a))
* **renovate:** reactive dashboard + config runs in one workflow ([#26](https://github.com/home-operations/downflate/issues/26)) ([bca80ae](https://github.com/home-operations/downflate/commit/bca80ae3483ad5edb2f485a2b119e4337e646a0c))
* skip release-please version-bump PRs in checks ([#27](https://github.com/home-operations/downflate/issues/27)) ([e241dee](https://github.com/home-operations/downflate/commit/e241dee0d3c5a03fa104c45e8628ad9aff97888a))
* wire govulncheck into mise and CI ([#44](https://github.com/home-operations/downflate/issues/44)) ([30d8054](https://github.com/home-operations/downflate/commit/30d8054130622485136f69eb59d509c099c428a9))


### Miscellaneous Chores

* add minimumGroupSize to Go toolchain configuration ([d493b59](https://github.com/home-operations/downflate/commit/d493b59750524cbc8bc7c77ce8708b137d1bcc61))
* **mise:** prune lockfile to used platforms ([#43](https://github.com/home-operations/downflate/issues/43)) ([db10f61](https://github.com/home-operations/downflate/commit/db10f613ed20afbb71501280edc7bd462fecda69))
* **mise:** Update mise tools ([#25](https://github.com/home-operations/downflate/issues/25)) ([d3f0103](https://github.com/home-operations/downflate/commit/d3f0103745d5ee78ff829970e686a3bfd034355a))
* **mise:** Update tool oxfmt (0.54.0 → 0.55.0) ([#4](https://github.com/home-operations/downflate/issues/4)) ([4fcbeb8](https://github.com/home-operations/downflate/commit/4fcbeb8161a738ed03062eead9a65b2ccbaa8a28))
* **mise:** Update tool oxfmt (0.55.0 → 0.56.0) ([#10](https://github.com/home-operations/downflate/issues/10)) ([66187b7](https://github.com/home-operations/downflate/commit/66187b7cb7e96b58e49e790a200568783eed1233))
* **mise:** Update tool oxfmt (0.56.0 → 0.57.0) ([#16](https://github.com/home-operations/downflate/issues/16)) ([e47573e](https://github.com/home-operations/downflate/commit/e47573ed6935140ebbe332961a4ca4b55a1acfd2))
* **mise:** Update tool oxfmt (0.60.0 → 0.61.0) ([#32](https://github.com/home-operations/downflate/issues/32)) ([a10f7b0](https://github.com/home-operations/downflate/commit/a10f7b0bd279e3258a3e7bbd9278cfedc0f32e15))
* **mise:** Update tool zizmor (1.25.2 → 1.26.1) ([#8](https://github.com/home-operations/downflate/issues/8)) ([f93fb99](https://github.com/home-operations/downflate/commit/f93fb99636c31da9b12c93691fbf2399e8f59d27))
* **mise:** Update tool zizmor (1.28.0 → 1.29.0) ([#41](https://github.com/home-operations/downflate/issues/41)) ([6c36f1d](https://github.com/home-operations/downflate/commit/6c36f1d7b43119f56647c2b62730f9314f7a665e))
* nit things ([99b7bf9](https://github.com/home-operations/downflate/commit/99b7bf9001f5bc0bba71c334be55e4a748d4e58a))
* **release-please:** standardize the release pull request title pattern ([#39](https://github.com/home-operations/downflate/issues/39)) ([993aeb0](https://github.com/home-operations/downflate/commit/993aeb05c3e92e4570e7ef022e3c62a891779ece))
* standardize release-please changelog sections ([#34](https://github.com/home-operations/downflate/issues/34)) ([581ffe2](https://github.com/home-operations/downflate/commit/581ffe2758d0b591c85abbea925ebd3823367436))
* to distroless image ([a6cb10b](https://github.com/home-operations/downflate/commit/a6cb10bb5dbd870a63c4b303328e0924062fcd97))
