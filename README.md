# 📘 LeetCode Daily Submission Fetcher

A Go script that periodically fetches **accepted LeetCode submissions**, retrieves their full code and problem description, and inserts them into a **Supabase database** via HTTP API. Perfect for maintaining a personal revision dashboard and tracking your daily coding progress.

---

## ✅ Features

- Retrieves **full code** and **problem description**
- Inserts data into two Supabase tables:
  - `leetcode_questions`: stores metadata and description
  - `leetcode_submissions`: stores each submission detail
- Automatically **upserts questions** to avoid duplicates
- Can be scheduled using **GitHub Actions** or **cron**

---

## 🛠️ Prerequisites

- Go 1.18+
- A [Supabase](https://supabase.com) project with the following tables:

### 📂 `leetcode_questions` Table Schema
```sql
create table leetcode_questions (
  slug text primary key,
  title text,
  description text
);
```

### 📂 `leetcode_submissions` Table Schema
```sql
create table leetcode_submissions (
  submission_id text primary key,
  question_slug text references leetcode_questions(slug),
  title text,
  submitted_at timestamptz,
  language text,
  status text,
  code text,
  description text
);
```

---

## 🔐 Environment Variables

Create a `.env` file or use GitHub Secrets:

```env
LEETCODE_USERNAME=your_leetcode_username
LEETCODE_SESSION=your_leetcode_session_cookie

SUPABASE_URL=https://your-project-id.supabase.co
SUPABASE_ANON_KEY=your_anon_key
```

> 💡 You can find `LEETCODE_SESSION` in browser cookies after logging into LeetCode.

---

## 🚀 Running the Script

```bash
go run main.go
```

Example output:

```
✅ Solved today: 2 problems

🔹 Two Sum
✅ Inserted to DB
🔹 Add Two Numbers
✅ Inserted to DB
```

---

## 🕒 Automate with GitHub Actions (Optional)

Use GitHub Actions to run the script daily:

```yaml
name: Fetch LeetCode Submissions Daily

on:
  schedule:
    - cron: '0 2 * * *' # Every day at 2 AM UTC
  workflow_dispatch:

jobs:
  fetch:
    runs-on: ubuntu-latest
    env:
      LEETCODE_USERNAME: ${{ secrets.LEETCODE_USERNAME }}
      LEETCODE_SESSION: ${{ secrets.LEETCODE_SESSION }}
      SUPABASE_URL: ${{ secrets.SUPABASE_URL }}
      SUPABASE_ANON_KEY: ${{ secrets.SUPABASE_ANON_KEY }}

    steps:
      - uses: actions/checkout@v3
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      - name: Run script
        run: go run main.go
```

---

## ✅ To Do
- [ ] Implement retry logic for failed GraphQL/API requests
- [ ] Auto-refresh/rotate `LEETCODE_SESSION` cookie (OAuth workaround?)

---

## ✨ Future Ideas

- Weekly challenge summaries via email or Telegram
- Leaderboard to track progress with friends
- AI-based suggestion system to revise past mistakes

---

Feel free to open issues or contribute ideas!
```
