# Go Realworld API

[realworld-apps/realworld](https://github.com/realworld-apps/realworld) の仕様に準拠したAPI

## 技術スタック

- 言語
  - Golang(1.25)
- WEbフレームワーク
  - Echo(v5)
- データベース
  - PostgreSQL 16
- DBドライバ
  - PGX/v5 (pgxpool)
- クエリ生成
  - sqlc
- 認証
  - JWT (golang-jwt/jwt v5)
- パスワードハッシュ
  - bcrypt
- マイグレーション
  - golang-migrate

## 要件定義(ユーザーストーリー、RealWorld仕様ベース)

### 記事(Article)

- ユーザーとして、記事を作成したい(タイトル・概要・本文・タグ)
- ユーザーとして、記事は自動生成される`slug`で識別・アクセスされてほしい
- 誰でも、公開されている記事の一覧・詳細を認証なしで見たい
- ユーザーとして、自分の記事を更新・削除したい(他人の記事は403)
- 誰でも、タグ・著者(username)・お気に入りしたユーザー(username)で記事を絞り込みたい
- 誰でも、記事一覧を`limit`/`offset`でページングしたい

### プロフィール・フォロー

- 誰でも、ユーザー名からプロフィール(`username`, `bio`, `image`, `following`)を見たい
- ユーザーとして、他のユーザーをフォロー/アンフォローしたい
- ログイン中のユーザーには、記事・プロフィールに「自分がフォロー中かどうか」を返してほしい

### フィード

- ユーザーとして、フォロー中のユーザーが書いた記事だけを新着順に見たい
- フォロー中のユーザーが0人の場合は空配列でよい

### お気に入り(Favorite)

- ユーザーとして、記事をお気に入り登録/解除したい
- 誰でも、記事ごとのお気に入り数(`favoritesCount`)を見たい
- ログイン中のユーザーには「自分が既にお気に入り済みか」(`favorited`)を返してほしい

### タグ

- 誰でも、存在するタグの一覧を見たい
- 記事作成・更新時に複数タグを指定したい(未知のタグは自動作成)

### コメント

- ユーザーとして、記事にコメントを投稿したい
- 誰でも、記事のコメント一覧を見たい(認証は任意)
- ユーザーとして、自分のコメントを削除したい(他人のコメントは403)

## エンドポイント

### エンドポイント一覧

| Method | Path | 認証 | 概要 |
|---|---|---|---|
| `POST` | `/api/users` | 不要 | ユーザー登録 |
| `POST` | `/api/users/login` | 不要 | ログイン |
| `GET` | `/api/user` | 必須 | 現在のユーザー情報取得 |
| `PUT` | `/api/user` | 必須 | 現在のユーザー情報更新 |
| `GET` | `/api/profiles/:username` | 任意 | プロフィール取得 |
| `POST` | `/api/profiles/:username/follow` | 必須 | フォロー |
| `DELETE` | `/api/profiles/:username/follow` | 必須 | アンフォロー |
| `GET` | `/api/articles?tag=&author=&favorited=&limit=20&offset=0` | 任意 | 記事一覧(絞り込み・ページネーション) |
| `GET` | `/api/articles/feed?limit=20&offset=0` | 必須 | フォロー中ユーザーの記事フィード |
| `GET` | `/api/articles/:slug` | 不要 | 記事詳細 |
| `POST` | `/api/articles` | 必須 | 記事作成 |
| `PUT` | `/api/articles/:slug` | 必須(著者のみ) | 記事更新 |
| `DELETE` | `/api/articles/:slug` | 必須(著者のみ) | 記事削除 |
| `GET` | `/api/articles/:slug/comments` | 任意 | コメント一覧 |
| `POST` | `/api/articles/:slug/comments` | 必須 | コメント投稿 |
| `DELETE` | `/api/articles/:slug/comments/:id` | 必須(投稿者のみ) | コメント削除 |
| `POST` | `/api/articles/:slug/favorite` | 必須 | お気に入り登録 |
| `DELETE` | `/api/articles/:slug/favorite` | 必須 | お気に入り解除 |
| `GET` | `/api/tags` | 不要 | タグ一覧 |

### レスポンス例

**ユーザー登録・ログイン**

```json
{
  "user": {
    "email": "taro@example.com",
    "token": "xxxxxx.yyyyyyy.zzzzzz",
    "username": "taro",
    "bio": null,
    "image": null
  }
}
```

**プロフィール**

```json
{
  "profile": {
    "username": "taro",
    "bio": "Goを勉強中",
    "image": null,
    "following": false
  }
}
```

**記事(単体)**

```json
{
  "article": {
    "slug": "go-context-nyumon",
    "title": "Goのcontext入門",
    "description": "context.Contextの使い方をまとめる",
    "body": "...",
    "tagList": ["go", "backend"],
    "createdAt": "2026-07-10T09:00:00.000Z",
    "updatedAt": "2026-07-10T09:00:00.000Z",
    "favorited": true,
    "favoritesCount": 5,
    "author": {
      "username": "taro",
      "bio": null,
      "image": null,
      "following": false
    }
  }
}
```

**記事一覧(ページネーション込み)**

```json
{
  "articles": [ /* Article配列 */ ],
  "articlesCount": 42
}
```

`limit`/`offset`はクエリパラメータで受け取り、`limit`デフォルトは20。`articlesCount`はフィルタ後の**総件数**(ページ内の件数ではない)を返す点に注意。

**コメント**

```json
{
  "comment": {
    "id": 1,
    "createdAt": "2026-07-10T09:00:00.000Z",
    "updatedAt": "2026-07-10T09:00:00.000Z",
    "body": "勉強になりました",
    "author": { "username": "jiro", "bio": null, "image": null, "following": false }
  }
}
```

**エラー(422)**

```json
{ "errors": { "body": ["can't be empty"] } }
```
