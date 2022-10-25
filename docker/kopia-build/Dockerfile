FROM golang:1.19-bullseye AS builder

ARG kopiaBuildCommit
ARG kopiaRepoOrg
ARG cgoFlag goExpFlag goExtFlag
ENV CGO_ENABLED=$cgoFlag GOEXPERIMENT=$goExpFlag GO_EXTLINK_ENABLED=$goExtFlag
RUN apt-get install git

WORKDIR /

RUN git clone https://github.com/${kopiaRepoOrg}/kopia.git

ENV GITHUB_REPOSITORY=https://github.com/${kopiaRepoOrg}/kopia

WORKDIR /kopia

# Build kopia binary from specific commit
RUN git checkout ${kopiaBuildCommit}
RUN GO111MODULE=on GOOS=linux GOARCH=amd64 go build -o kopia \
  -ldflags="-X github.com/kopia/kopia/repo.BuildVersion=$(git show --no-patch --format='%cs-%h') \
            -X github.com/kopia/kopia/repo.BuildInfo=$(git show --no-patch --format='%cI-%H')-${kopiaBuildCommit} \
            -X github.com/kopia/kopia/repo.BuildGitHubRepo=${kopiaRepoOrg}" .

RUN adduser kopia && addgroup kopia kopia
USER kopia:kopia

COPY --chown=kopia . /kopia

FROM debian:bullseye

WORKDIR /kopia

# Add CA certs
RUN apt-get update && apt-get -y install ca-certificates && \
  rm -rf /var/cache/apk/* && \
  adduser kopia && addgroup kopia kopia && \
  chown kopia /kopia

USER kopia:kopia

ENTRYPOINT [ "/kopia/kopia" ]

LABEL imageVersion="${imageVersion}"
LABEL kopiaBuildCommit="${kopiaBuildCommit}"

COPY --from=builder --chown=kopia:kopia /kopia/kopia /kopia/kopia
