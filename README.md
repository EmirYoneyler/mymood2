# mymood

Kullanıcıların günlerine 1-10 arası puan verdiği, arkadaş ekleyip onların gün puanlarını görebildiği sosyal bir mood-tracking platformu.

## Teknoloji Yığını

- **Backend:** Go + [Fiber](https://gofiber.io/)
- **Veritabanı:** PostgreSQL ([Neon.tech](https://neon.tech) ücretsiz katman)
- **DB Erişimi:** `pgx/v5` ile el yazımı, parametreli SQL sorguları (ORM yok)
- **Migration:** [golang-migrate](https://github.com/golang-migrate/migrate) — SQL dosyaları binary'ye `go:embed` ile gömülü, uygulama açılışında otomatik çalışır
- **Frontend:** Go `html/template` + [HTMX](https://htmx.org/) + [Tailwind CSS](https://tailwindcss.com/) (Play CDN)
- **Auth:** JWT (`golang-jwt/jwt/v5`) + `bcrypt`, HttpOnly + Secure + SameSite=Strict cookie tabanlı session
- **Validation:** `go-playground/validator`

Tek bir Go binary'si olarak derlenir; ayrı bir frontend build/deploy adımı yoktur.

## Klasör Yapısı

```
mymood/
├── cmd/server/        # main.go — uygulama girişi
├── internal/
│   ├── config/        # env değişkenleri
│   ├── database/      # pgx pool + migration runner
│   ├── models/         # User, MoodEntry, Friendship, FeedEntry struct'ları
│   ├── handlers/        # auth, mood, friend, feed, profile
│   ├── middleware/        # JWT auth middleware, premium gate (Faz 2 için hazır)
│   └── repository/        # parametreli SQL sorguları
├── web/
│   ├── templates/      # layouts + pages
│   └── static/css       # app.css (Tailwind ile birlikte özel stiller)
├── migrations/         # SQL migration dosyaları (embed)
├── Dockerfile
└── .env.example
```

## Yerel Geliştirme

1. Go 1.26+ kurulu olmalı.
2. Bir PostgreSQL veritabanı hazırla (yerel ya da Neon.tech).
3. `.env.example`'ı `.env` olarak kopyala ve `DATABASE_URL` / `JWT_SECRET` değerlerini doldur:

   ```bash
   cp .env.example .env
   ```

4. Bağımlılıkları indir ve çalıştır:

   ```bash
   go mod download
   go run ./cmd/server
   ```

   Uygulama açılışta migration'ları otomatik uygular ve `:3000` portunda (veya `PORT` env değişkeninde belirtilen portta) dinlemeye başlar.

5. `DATABASE_URL` boşsa uygulama sadece `/healthz` endpoint'i ile açılır (migration/DB bağlantısı atlanır) — bu, veritabanı olmadan hızlıca derleme/boot kontrolü yapmaya yarar.

## Doğrulama

```bash
go build ./...
go vet ./...
gofmt -l .
```

## Deployment (ücretsiz katman)

1. Kodu GitHub'a push et.
2. [Neon.tech](https://neon.tech)'de ücretsiz proje oluştur, `DATABASE_URL` connection string'ini al (sslmode=require ile).
3. [Render.com](https://render.com)'da "New Web Service" oluştur, GitHub reponu bağla, build yöntemi olarak `Dockerfile`'ı seç, instance type **Free**.
4. Render → Environment kısmına `DATABASE_URL` ve `JWT_SECRET` değerlerini ekle (`APP_ENV=production` da eklenmesi önerilir).
5. Deploy sırasında uygulama açılışta migration'ları otomatik çalıştırır — ayrı bir migration adımına gerek yoktur.
6. Render'ın ücretsiz katmanı 15 dakika hareketsizlikten sonra uyur; ilk istekte birkaç saniyelik gecikme olur (demo/portföy projesi için kabul edilebilir).
7. `mymood.com` gibi bir domain alındığında DNS ile Render servisine yönlendirilebilir; domain ücretli, hosting ücretsiz kalmaya devam eder.

## Faz 2 (henüz implement edilmedi, mimaride yer ayrılmış)

- **Reklam:** Feed, her giriş bağımsız bir kart olarak render edildiği için (`web/templates/pages/feed.html`), aralarına bir ad-slot komponenti eklemek mevcut yapıyı bozmadan mümkün.
- **Abonelik:** `users.is_premium` kolonu zaten mevcut. İleride bir `subscriptions` tablosu (plan, status, started_at, expires_at) eklenecek. `internal/middleware/premium.go` içinde, henüz hiçbir route'a bağlanmamış bir `RequirePremium` middleware'i hazır bekliyor — premium bir özellik eklendiğinde sadece route tanımına eklenmesi yeterli olacak.

## Güvenlik Notları

- Şifreler `bcrypt` ile hash'lenir.
- Session JWT'leri HttpOnly + SameSite=Strict cookie içinde tutulur (üretimde `Secure` bayrağı `APP_ENV=production` ile otomatik açılır).
- Tüm form girdileri sunucu tarafında `validator` ile doğrulanır.
- `/login` ve `/register` endpoint'leri dakikada 10 istekle sınırlıdır (rate limiting).
- Tüm SQL sorguları parametreli çalışır (pgx), SQL injection riski yoktur.
