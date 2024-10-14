# Tanuki

`tanuki` is a **dead simple gitlab search in terminal**, inspired by [gitlab-search](https://github.com/phillipj/gitlab-search)

Written in Go with love.

## Prerequisites

- A personal GitLab access token with the `read_api` scope.

## Installation

### Via go
```bash
go install github.com/yan-aint-nickname/tanuki
```

### Via homebrew
Note: !It's not always the latest version!
```bash
brew tap yan-aint-nickname/tanuki
brew install tanuki
```

## Usage

```bash
tanuki --token="<your token goes here>" --server="<your server goes here>" search --group="<your group goes here>" "<search string>"
```

## Note:
It's under active development
