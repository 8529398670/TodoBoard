# vendor/

Third-party browser libraries live here as plain files, downloaded once and
served by this app.

Nothing in this project is compiled, bundled, or transpiled, and nothing is
fetched from a CDN at runtime. Two reasons that is worth the small hassle of
downloading a file:

- **The page keeps working.** A CDN outage, a DNS block, or a corporate proxy
  cannot take a self-hosted file away, and the version cannot change under you
  because someone re-tagged a release.
- **The Content-Security-Policy stays tight.** `server/middleware` sets
  `script-src 'self'`, so a `<script src="https://cdn...">` is refused by the
  browser. That is the policy working, not a bug -- vendor the file instead of
  loosening it.

Add one with the helper, which downloads the file and prints the tag to paste:

```bash
./scripts/vendor.sh https://cdnjs.cloudflare.com/ajax/libs/htmx.org/2.0.4/htmx.min.js
```

Then reference it from your HTML:

```html
<script src="/vendor/htmx.min.js"></script>
```

Commit what lands here. It is part of the app, not a build artifact.

## What is here

| Library | Version | Files | Used for |
|---|---|---|---|
| [marked](https://marked.js.org) | 15.0.12 | `marked/marked.min.js` | Markdown → HTML (`js/markdown.js`) |
| [DOMPurify](https://github.com/cure53/DOMPurify) | 3.2.6 | `dompurify/purify.min.js` | sanitising that HTML before it touches the page |
| [KaTeX](https://katex.org) | 0.16.22 | `katex/katex.min.{js,css}`, `katex/fonts/*.woff2`, `katex/contrib/mhchem.min.js` | `$…$` / `$$…$$` math, and `\ce{}` / `\pu{}` chemistry via mhchem |
| [SortableJS](https://sortablejs.github.io/Sortable/) | 1.15.6 | `sortable/Sortable.min.js` | drag and drop of cards, lists, checklist items (touch included) |
| [html-to-image](https://github.com/bubkoo/html-to-image) | 1.11.13 | `html-to-image/html-to-image.js` | exporting a card as a PNG |

sha256 of each file as downloaded:

```
3e7e7d7feb3e5d58cb6c804f68ab5c24cc7e5eb6270fd6e5cbb9124739217d0c  marked/marked.min.js
89e1fa7647cb495370d3a997ace4387f5d15d9f4c5af12352c53daa400956287  dompurify/purify.min.js
e8d885505949f3a5f4abdd5dd0d53696bd1371ad26ffbf4f310dcd77c8cdae89  katex/katex.min.js
19095127357ed6d29fe0a63a6b000c913a89f7f1963b765dd3715e97c9852e75  katex/katex.min.css
9f87e5e9c384a160472d0045035a8641f6013358eddb3ece708634a50f946a40  katex/contrib/mhchem.min.js
6d0a831fc19b4bae851797ad3393157e861afb7862459c11226359b27e2c4337  sortable/Sortable.min.js
a90b42909d80964269ef6d5f3d1e4a5a7e2a4c263a5d2a76a9e7151901343262  html-to-image/html-to-image.js
```

Only the `.woff2` KaTeX fonts are vendored; every browser this app supports
picks those first, so the `.woff`/`.ttf` fallbacks the stylesheet lists are
never requested.

KaTeX is always called as `katex.render( tex , element )`, never
`renderToString`: `render` builds DOM nodes and sets their styles through the
CSSOM, which the `style-src 'self'` policy allows, whereas the string form
produces `style="..."` attributes the policy would strip.
