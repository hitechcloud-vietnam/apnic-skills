# About

Background information about the `apnic-skills` project: how it has evolved, the terms under which you can use it, and how to contribute.

## Sub-pages

- [Changelog](changelog.md) — Key version changes, drawn from the git history.
- [License](license.md) — The MIT License under which the project is distributed.
- [Contributing](contributing.md) — Development setup, the 100% test-coverage requirement, and the pull-request process.

## Project at a Glance

| | |
|---|---|
| **Language** | Go 1.25 |
| **Module** | `github.com/hitechcloud-vietnam/apnic-skills` |
| **License** | MIT |
| **Test coverage** | 100% (SDK statement coverage; CLI named functions 100%) |
| **Repository** | [hitechcloud-vietnam/apnic-skills](https://github.com/hitechcloud-vietnam/apnic-skills) |
| **Documentation** | [hitechcloud-vietnam.github.io/apnic-skills](https://hitechcloud-vietnam.github.io/apnic-skills/) |

## Release Timeline

```mermaid
gantt
    title Project history (by commit date)
    dateFormat  YYYY-MM-DD
    axisFormat  %Y-%m

    section Bootstrap
    Initial commit                                   :milestone, init, 2025-02-24, 0d

    section Hardening
    .gitignore + submodule sync                      :task1, 2026-01-29, 1d
    100% test coverage for all SDK APIs              :milestone, cov, 2026-06-28, 0d

    section BGP expansion
    Thyme BGP models + parsers (5 files)             :task2, 2026-07-04, 1d
    Multi-source (au/hk) support                     :task3, 2026-07-04, 1d
    5 BGP fetchers + unit tests                      :task4, 2026-07-04, 1d
    5 CLI subcommands + --bgp-source flag            :task5, 2026-07-04, 1d
    used-autnums panic fix + bad-prefixes truncation :task6, 2026-07-04, 1d

    section Tooling
    Anti-scraping + IRR/REx/RRDP/telemetry work      :task7, 2026-07-04, 1d
    Merge PR #4 (thyme-bgp-additional)               :milestone, pr4, 2026-07-04, 0d
```

See the [Changelog](changelog.md) for the detailed, dated feature list.
