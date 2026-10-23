# Documentation

This project contains the documentation sources for the [ActiveMQ Artemis (MQP)](https://activemq.pnet.ch/) platform, rendered via [Backstage TechDocs](https://backstage.io/docs/features/techdocs/).

## Run locally

Use [techdocs-cli](https://backstage.io/docs/features/techdocs/cli) to preview the docs exactly as they appear in Backstage.

**Install:**

```shell
npm install -g @techdocs/cli
```

**Serve using Docker**

```shell
techdocs-cli serve
```

Opens a preview at <http://localhost:3000/docs/default/resource/activemq/>.

## Generate section indexes

Each top-level nav section (e.g. `HowTo`, `Concepts`, `Infrastructure`) needs an `index.md` so the sidebar entry is 
clickable. These are produced by `generate-section-indexes.py` and written into `docs/<section>/index.md`. In CI the 
`create-section-indexes` job runs it automatically before the TechDocs build.

Run it locally before `techdocs-cli serve`:

```shell
docker run --rm -v "$PWD:/content" -w /content middleware-docker-local.repo.pnet.ch/pf/middleware/experimental/techdocs python generate-section-indexes.py
```

Existing `index.md` files (e.g. `docs/index.md`, blog) are left untouched.
