package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	bolt "go.etcd.io/bbolt"
	"gopkg.in/gomail.v2"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Watch    []WatchEntry `yaml:"watch"`
	Channels Channels     `yaml:"channels"`
}
type WatchEntry struct {
	ID              string `yaml:"id"`
	Name            string `yaml:"name"`
	Language        string `yaml:"language"`
	CurrentVersion  string `yaml:"current_version"`
	DeprecationDate string `yaml:"deprecation_date"`
	Recommended     string `yaml:"recommended"`
}
type Channels struct {
	Email     EmailChannel `yaml:"email"`
	Discord   struct{ Enabled bool `yaml:"enabled"` } `yaml:"discord"`
	Messenger struct{ Enabled bool `yaml:"enabled"` } `yaml:"messenger"`
}
type EmailChannel struct {
	From            string `yaml:"from"`
	SubjectTemplate string `yaml:"subject_template"`
}

const (
	dbPath       = "notifier.db"
	templateFile = "templates/email.html"
)

func main() {
	runOnce := flag.Bool("run-once", false, "run checks once and exit")
	cfgPath := flag.String("config", "config.yaml", "path to config.yaml")
	flag.Parse()

	// load config and template
	cfg, err := loadConfig(*cfgPath)
	if err != nil {
		log.Fatal("load config:", err)
	}
	tplBytes, err := os.ReadFile(templateFile)
	if err != nil {
		log.Fatal("load template:", err)
	}
	tpl := template.Must(template.New("email").Parse(string(tplBytes)))

	// open DB
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Fatal("open db:", err)
	}
	defer db.Close()

	// Start simple HTTP health server in background (port from $PORT or 8080)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	})
	go func() {
		log.Println("health server listening on :" + port)
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Fatal("health server:", err)
		}
	}()

	// If run-once mode, run check and exit
	if *runOnce {
		if err := runCheck(cfg, db, tpl); err != nil {
			log.Fatal("runCheck:", err)
		}
		log.Println("run-once complete")
		return
	}

	// Continuous mode: run immediately then every 24 hours
	if err := runCheck(cfg, db, tpl); err != nil {
		log.Println("initial runCheck:", err)
	}
	ticker := time.NewTicker(24 * time.Hour)
	for range ticker.C {
		if err := runCheck(cfg, db, tpl); err != nil {
			log.Println("runCheck:", err)
		}
	}
}

func loadConfig(p string) (*Config, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func runCheck(cfg *Config, db *bolt.DB, tpl *template.Template) error {
	now := time.Now().UTC()
	for _, w := range cfg.Watch {
		depDate, err := time.Parse("2006-01-02", w.DeprecationDate)
		if err != nil {
			log.Println("invalid date for", w.ID, err)
			continue
		}
		daysLeft := int(depDate.Sub(now).Hours() / 24)
		yearsLeft := float64(daysLeft) / 365.0

		// 365..912 days === 1..2.5 years
		if daysLeft >= 365 && daysLeft <= 912 {
			key := "sent:" + w.ID + ":" + depDate.Format("2006-01-02")
			sent, err := alreadySent(db, key)
			if err != nil {
				log.Println("db check:", err)
				continue
			}
			if sent {
				log.Println("already sent for", w.ID)
				continue
			}

			data := map[string]interface{}{
				"Name":           w.Name,
				"CurrentVersion": w.CurrentVersion,
				"DaysLeft":       daysLeft,
				"YearsLeft":      fmt.Sprintf("%.1f", yearsLeft),
				"Recommended":    w.Recommended,
			}

			var bodyBuf bytes.Buffer
			if err := tpl.Execute(&bodyBuf, data); err != nil {
				log.Println("tpl exec:", err)
				continue
			}
			htmlBody := bodyBuf.String()
			subj := cfg.Channels.Email.SubjectTemplate
			if subj == "" {
				subj = fmt.Sprintf("%s will be deprecated in %.1f years", w.Name, yearsLeft)
			}
			subj = replacePlaceholders(subj, data)

			// Prefer SMTP if configured
			if os.Getenv("SMTP_HOST") != "" {
				from := os.Getenv("SMTP_FROM_EMAIL")
				if from == "" {
					from = os.Getenv("SMTP_USERNAME")
				}
				if err := sendSMTP(os.Getenv("NOTIFY_TO_EMAIL"), subj, htmlBody, subj, from); err != nil {
					log.Println("smtp send err:", err)
				} else {
					log.Println("smtp sent for", w.ID)
				}

			} else if os.Getenv("MAILERSEND_API_KEY") != "" && os.Getenv("NOTIFY_TO_EMAIL") != "" {
				if err := sendMailerSendHTTP(os.Getenv("MAILERSEND_API_KEY"), os.Getenv("MAILERSEND_FROM_EMAIL"), os.Getenv("NOTIFY_TO_EMAIL"), subj, htmlBody); err != nil {
					log.Println("mail send err:", err)
				} else {
					log.Println("mail sent for", w.ID)
				}
			} else {
				log.Println("no mail config found, skipping email")
			}

			// Discord
			if cfg.Channels.Discord.Enabled && os.Getenv("DISCORD_WEBHOOK_URL") != "" {
				msg := fmt.Sprintf("%s will be deprecated in approx %.1f years (%d days). Recommended: %s", w.Name, yearsLeft, daysLeft, w.Recommended)
				if err := sendDiscordWebhook(os.Getenv("DISCORD_WEBHOOK_URL"), msg); err != nil {
					log.Println("discord err:", err)
				} else {
					log.Println("discord sent for", w.ID)
				}
			}

			// Messenger
			if cfg.Channels.Messenger.Enabled && os.Getenv("FB_PAGE_ACCESS_TOKEN") != "" && os.Getenv("FB_RECIPIENT_ID") != "" {
				msg := fmt.Sprintf("%s will be deprecated in approx %.1f years (%d days). Recommended: %s", w.Name, yearsLeft, daysLeft, w.Recommended)
				if err := sendFacebookMessenger(os.Getenv("FB_PAGE_ACCESS_TOKEN"), os.Getenv("FB_RECIPIENT_ID"), msg); err != nil {
					log.Println("fb err:", err)
				} else {
					log.Println("facebook messenger sent for", w.ID)
				}
			}

			if err := markSent(db, key); err != nil {
				log.Println("markSent err:", err)
			}
		} else {
			log.Printf("skip %s: daysLeft=%d\n", w.ID, daysLeft)
		}
	}
	return nil
}

func alreadySent(db *bolt.DB, key string) (bool, error) {
	var found bool
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("sent"))
		if b == nil {
			return nil
		}
		if b.Get([]byte(key)) != nil {
			found = true
		}
		return nil
	})
	return found, err
}
func markSent(db *bolt.DB, key string) error {
	return db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("sent"))
		if err != nil {
			return err
		}
		return b.Put([]byte(key), []byte(time.Now().Format(time.RFC3339)))
	})
}

func replacePlaceholders(s string, data map[string]interface{}) string {
	out := s
	for k, v := range data {
		ph := "{{." + k + "}}"
		out = string(bytes.ReplaceAll([]byte(out), []byte(ph), []byte(fmt.Sprint(v))))
	}
	return out
}

// --- networking senders ---

func sendSMTP(to, subject, htmlBody, textBody, from string) error {
	host := os.Getenv("SMTP_HOST")
	port := 587
	if p := os.Getenv("SMTP_PORT"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			port = v
		}
	}
	user := os.Getenv("SMTP_USERNAME")
	pass := os.Getenv("SMTP_PASSWORD")

	m := gomail.NewMessage()
	if from != "" {
		m.SetHeader("From", m.FormatAddress(from, os.Getenv("SMTP_FROM_NAME")))
	} else {
		// fallback to username
		m.SetHeader("From", user)
	}
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", textBody)
	m.AddAlternative("text/html", htmlBody)

	d := gomail.NewDialer(host, port, user, pass)
	// gomail uses STARTTLS for port 587 automatically
	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("smtp send failed: %w", err)
	}
	return nil
}

func sendMailerSendHTTP(apiKey, from, to, subject, htmlBody string) error {
	payload := map[string]interface{}{
		"from": map[string]string{"email": from, "name": "Deprecation Notifier"},
		"to":   []map[string]string{{"email": to}},
		"subject": subject,
		"html":    htmlBody,
		"text":    subject,
	}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "https://api.mailersend.com/v1/email", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("mailersend status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func sendDiscordWebhook(url, content string) error {
	payload := map[string]string{"content": content}
	b, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func sendFacebookMessenger(pageToken, recipientID, text string) error {
	url := "https://graph.facebook.com/v16.0/me/messages?access_token=" + pageToken
	payload := map[string]interface{}{
		"recipient": map[string]string{"id": recipientID},
		"message":   map[string]string{"text": text},
	}
	b, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("fb status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}
