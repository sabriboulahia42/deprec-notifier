# 1) Fetch remote refs and create a merge branch based on origin/main
git fetch origin
git checkout -b merge/sync-laptop origin/main

# 2) Merge your laptop branch into this new branch (allow unrelated histories)
git merge --allow-unrelated-histories sync/laptop -m "Merge sync/laptop into main (allow unrelated histories)"

# 3) If there are conflicts, list them, resolve in your editor, then mark resolved:
git status --porcelain
git diff --name-only --diff-filter=U

# After resolving conflicts in the files, stage and finish the merge:
git add -A
git commit -m "Resolve merge conflicts (merge/sync-laptop)"

# 4) Inspect the merged commits (optional)
git log --oneline origin/main..merge/sync-laptop
git diff --name-only origin/main...merge/sync-laptop

# 5) Push the merge branch to GitHub
git push -u origin merge/sync-laptop

# 6) Open the compare page in your browser to create & merge the PR manually
Start-Process "https://github.com/sabriboulahia42/deprec-notifier/compare/main...merge/sync-laptop?expand=1"