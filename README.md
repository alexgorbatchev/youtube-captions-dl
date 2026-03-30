# youtube-captions-dl

A small CLI utility that prints YouTube captions as plain text to stdout.

## Behavior

- accepts exactly one argument: a YouTube URL
- prints only caption text to stdout
- removes timestamps and inline formatting
- prefers a human-created caption track when available
- falls back to the first auto-generated caption track
- caches the final plain-text transcript under the XDG cache directory

## Cache location

This tool uses the XDG cache convention directly:

- `$XDG_CACHE_HOME/youtube-captions-dl` when `XDG_CACHE_HOME` is set
- `$HOME/.cache/youtube-captions-dl` otherwise

## Install

```bash
go install github.com/alexgorbatchev/youtube-captions-dl@latest
```

## Usage

```bash
youtube-captions-dl 'https://www.youtube.com/watch?v=...'
```

## Build locally

```bash
just build
./bin/youtube-captions-dl 'https://www.youtube.com/watch?v=...'
```

## Important limitation

This tool uses YouTube's web/player endpoints because the official YouTube Data API caption download endpoint requires authorization and edit permission on the video. That means there is no stable official API for arbitrary public-video transcript downloads.

