# syntax=docker/dockerfile:1
# SPDX-License-Identifier: MIT

# RepoKeeper as an MCP server, for docker-based MCP client configurations.
#
# This image is built by GoReleaser's dockers_v2 block from binaries it has
# already compiled for every target platform -- there is deliberately no build
# stage here. Compiling inside the image would produce a *different* binary from
# the one in the release archives, breaking the property that every channel is
# fed from one build: the same bytes that were signed, notarized, checksummed
# and attested are the bytes that ship here.
#
# The build context is laid out per platform by GoReleaser:
#   linux/amd64/repokeeper
#   linux/arm64/repokeeper

# Alpine, not distroless.
#
# sting's equivalent image uses gcr.io/distroless/static, because its server
# only makes HTTPS calls. RepoKeeper cannot: internal/gitx shells out to the
# installed git binary for every inspection it performs ("It shells out to the
# installed git binary" -- internal/gitx/gitx.go). A distroless image has no
# git, so every tool would fail at exec.LookPath.
#
# Pinned by digest so a rebuild cannot silently pick up a different base. Note
# that this pins the base only: the git and ca-certificates package versions
# installed below float with the Alpine repository, which is stated here rather
# than implied.
FROM alpine:3.24.1@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b

# git is a hard runtime requirement, not a convenience. ca-certificates is
# needed for any operation that reaches a remote over HTTPS.
RUN apk add --no-cache git ca-certificates

# Git refuses to operate on a repository directory owned by a different uid than
# the running process ("detected dubious ownership", git >= 2.35.2). A workspace
# bind-mounted from the host is owned by the host user -- typically uid 1000 --
# while this image runs as 65532, so *every* git invocation would fail with
# exit 128 and the user would see a git internals error rather than anything
# RepoKeeper explains.
#
# Measured, not assumed: at uid 65532 against a bind-mounted repository, git
# 2.55.0 exits 128 without this setting and 0 with it.
#
# safe.directory=* is discouraged on a shared multi-user host, because it lets
# git act on repositories owned by other users. That reasoning does not apply
# here: the container's entire filesystem view is whatever the user chose to
# mount, so there is no foreign repository to be tricked into trusting.
RUN git config --system safe.directory '*'

# Set by buildx for each platform in the manifest.
ARG TARGETPLATFORM

COPY --chmod=0755 ${TARGETPLATFORM}/repokeeper /usr/local/bin/repokeeper

# uid/gid 65532 matches the conventional non-root user used across Skaphos
# images. RepoKeeper needs no privileges: it reads repository working trees and,
# only when explicitly asked and given a read-write mount, updates them.
#
# For read-write use, run with --user "$(id -u):$(id -g)" so files git creates
# land with the host user's ownership. Reads work at any uid thanks to the
# safe.directory setting above.
USER 65532:65532

# No WORKDIR, deliberately.
#
# RepoKeeper finds its registry by walking upward from the working directory to
# the nearest .repokeeper.yaml, which is what lets the CLI serve several
# purpose-specific workspace roots that self-select by position. A container's
# working directory is fixed before any tool call, so that behavior cannot be
# reproduced. A default WORKDIR would be correct only for users who happen to
# mount there, and a default that is right for some users and silently wrong for
# others is worse than requiring the workspace to be named.
#
# Mount the workspace at its identical host path and name the registry
# explicitly, one configured server entry per workspace root:
#
#   docker run -i --rm \
#     -v /home/me/work/skaphos:/home/me/work/skaphos:ro \
#     ghcr.io/skaphos/repokeeper:VERSION \
#     mcp --config /home/me/work/skaphos/.repokeeper.yaml
#
# Mercurial is not installed. RepoKeeper's Mercurial backend is experimental and
# opt-in per command; the container supports Git only, and says so rather than
# failing at exec.LookPath.

# With no arguments the container is an MCP server speaking stdio, which is what
# a docker-based MCP client configuration expects. Other subcommands remain
# reachable by overriding the command.
ENTRYPOINT ["/usr/local/bin/repokeeper"]
CMD ["mcp"]
