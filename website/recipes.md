# Recipes

Practical, copy-paste workflows for common erdn-lang integrations.

## GitHub Actions: Comment ERD render results on pull requests

If you want reviewers to see ERD rendering results directly on a PR, add a workflow like this:

```yaml
name: ERD PR Comment

on:
  pull_request:
    types: [opened, synchronize, reopened]
    paths:
      - "**/*.erdn"

permissions:
  contents: read
  pull-requests: write

jobs:
  render-and-comment:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.22"

      - name: Install erdn
        run: go install github.com/headercat/erdn-lang/cmd/erdn@latest

      - name: Render changed .erdn files
        id: render
        shell: bash
        run: |
          set -euo pipefail

          mkdir -p .erdn-render
          marker="<!-- erdn-render-results -->"

          changed_files="$(git diff --name-only origin/${{ github.base_ref }}...HEAD -- '*.erdn')"
          if [[ -z "${changed_files}" ]]; then
            {
              echo "${marker}"
              echo "## ERD render results"
              echo ""
              echo "No \`.erdn\` file changes were detected in this pull request."
            } > .erdn-render/comment.md
            echo "has_svg=false" >> "$GITHUB_OUTPUT"
            echo "render_failed=false" >> "$GITHUB_OUTPUT"
            exit 0
          fi

          render_failed=0
          {
            echo "${marker}"
            echo "## ERD render results"
            echo ""
          } > .erdn-render/comment.md

          while IFS= read -r file; do
            [[ -z "${file}" ]] && continue
            out=".erdn-render/$(echo "${file}" | tr '/ ' '__').svg"

            if erdn render "${file}" --out "${out}" > .erdn-render/render.log 2>&1; then
              echo "- ✅ \`${file}\` rendered to \`$(basename "${out}")\`" >> .erdn-render/comment.md
            else
              render_failed=1
              {
                echo "- ❌ \`${file}\` failed to render"
                echo "  <details><summary>Show error</summary>"
                echo ""
                echo "  \`\`\`text"
                sed 's/^/  /' .erdn-render/render.log
                echo "  \`\`\`"
                echo "  </details>"
              } >> .erdn-render/comment.md
            fi
          done <<< "${changed_files}"

          if ls .erdn-render/*.svg >/dev/null 2>&1; then
            echo "" >> .erdn-render/comment.md
            echo "Download rendered SVG artifacts from this run:" >> .erdn-render/comment.md
            echo "https://github.com/${{ github.repository }}/actions/runs/${{ github.run_id }}" >> .erdn-render/comment.md
            echo "has_svg=true" >> "$GITHUB_OUTPUT"
          else
            echo "has_svg=false" >> "$GITHUB_OUTPUT"
          fi

          if [[ "${render_failed}" -eq 1 ]]; then
            echo "render_failed=true" >> "$GITHUB_OUTPUT"
          else
            echo "render_failed=false" >> "$GITHUB_OUTPUT"
          fi

      - name: Upload SVG artifacts
        if: steps.render.outputs.has_svg == 'true'
        uses: actions/upload-artifact@v4
        with:
          name: erd-diagrams
          path: .erdn-render/*.svg

      - name: Create or update PR comment
        if: always()
        uses: actions/github-script@v7
        with:
          script: |
            const fs = require("fs");
            const marker = "<!-- erdn-render-results -->";
            const body = fs.readFileSync(".erdn-render/comment.md", "utf8");
            const { owner, repo } = context.repo;
            const issue_number = context.issue.number;

            const comments = await github.paginate(
              github.rest.issues.listComments,
              { owner, repo, issue_number, per_page: 100 }
            );
            const existing = comments.find(
              (c) => c.user?.type === "Bot" && c.body?.includes(marker)
            );

            if (existing) {
              await github.rest.issues.updateComment({
                owner,
                repo,
                comment_id: existing.id,
                body
              });
            } else {
              await github.rest.issues.createComment({
                owner,
                repo,
                issue_number,
                body
              });
            }

      - name: Fail workflow if render failed
        if: steps.render.outputs.render_failed == 'true'
        run: exit 1
```
