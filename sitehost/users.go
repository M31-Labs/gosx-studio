package sitehost

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// users.go is who can sign in, and as what.
//
// Three roles are enough for a small team: an owner, who can do anything;
// admins, who can do anything but hand the site to someone else; and
// editors, who write and publish but do not touch settings, payments, the
// domain, or people. Passwords are bcrypt hashes, sessions are random tokens
// stored hashed, invites are one-use links, and a second factor is a
// standard authenticator code (TOTP). Nothing here needs a third party.

const (
	roleOwner  = "owner"
	roleAdmin  = "admin"
	roleEditor = "editor"

	sessionCookie  = "gosx_session"
	sessionTTL     = 30 * 24 * time.Hour
	inviteTTL      = 7 * 24 * time.Hour
	totpPeriod     = 30
	totpDigits     = 6
	minPasswordLen = 10
)

// passwordCost is bcrypt's work factor; tests lower it.
var passwordCost = bcrypt.DefaultCost

// User is one person who can sign in.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	Role         string    `json:"role"`
	PasswordHash string    `json:"passwordHash"`
	TOTPSecret   string    `json:"totpSecret,omitempty"`
	Created      time.Time `json:"created"`
	LastLogin    time.Time `json:"lastLogin,omitempty"`
}

func (u User) roleLabel() string {
	switch u.Role {
	case roleOwner:
		return "Owner"
	case roleAdmin:
		return "Admin"
	default:
		return "Editor"
	}
}

// Invite is a link that turns into a user once.
type Invite struct {
	TokenHash string    `json:"tokenHash"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Expires   time.Time `json:"expires"`
	InvitedBy string    `json:"invitedBy,omitempty"`
}

// Session is one signed-in browser.
type Session struct {
	TokenHash string    `json:"tokenHash"`
	UserID    string    `json:"userId"`
	Created   time.Time `json:"created"`
	Expires   time.Time `json:"expires"`
}

func normalizeRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case roleOwner:
		return roleOwner
	case roleAdmin:
		return roleAdmin
	default:
		return roleEditor
	}
}

// roleRank orders roles so "at least admin" is one comparison.
func roleRank(role string) int {
	switch normalizeRole(role) {
	case roleOwner:
		return 3
	case roleAdmin:
		return 2
	default:
		return 1
	}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validEmail(email string) bool {
	at := strings.Index(email, "@")
	return at > 0 && at < len(email)-3 && strings.Contains(email[at:], ".") && !strings.ContainsAny(email, " <>\n")
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func newToken() string { return randomHex(24) }

// ---------- the store ----------

type userStore struct {
	mu       sync.Mutex
	path     string
	loaded   bool
	users    []User
	invites  []Invite
	sessions []Session
}

func newUserStore(path string) *userStore { return &userStore{path: path} }

func (o Options) usersPath() string {
	if o.DataPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(o.DataPath), "users.json")
}

type userFile struct {
	Users    []User    `json:"users"`
	Invites  []Invite  `json:"invites,omitempty"`
	Sessions []Session `json:"sessions,omitempty"`
}

func (s *userStore) loadLocked() {
	if s.loaded {
		return
	}
	s.loaded = true
	if s.path == "" {
		return
	}
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var file userFile
	if json.Unmarshal(raw, &file) == nil {
		s.users, s.invites, s.sessions = file.Users, file.Invites, file.Sessions
	}
}

func (s *userStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	now := timeNow().UTC()
	invites := s.invites[:0]
	for _, invite := range s.invites {
		if invite.Expires.After(now) {
			invites = append(invites, invite)
		}
	}
	s.invites = invites
	sessions := s.sessions[:0]
	for _, session := range s.sessions {
		if session.Expires.After(now) {
			sessions = append(sessions, session)
		}
	}
	s.sessions = sessions
	raw, err := json.MarshalIndent(userFile{s.users, s.invites, s.sessions}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	temp := s.path + ".tmp"
	if err := os.WriteFile(temp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(temp, s.path)
}

func (s *userStore) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	return len(s.users)
}

func (s *userStore) list() []User {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	out := make([]User, len(s.users))
	copy(out, s.users)
	sort.SliceStable(out, func(i, j int) bool {
		if roleRank(out[i].Role) != roleRank(out[j].Role) {
			return roleRank(out[i].Role) > roleRank(out[j].Role)
		}
		return out[i].Created.Before(out[j].Created)
	})
	return out
}

func (s *userStore) byID(id string) (User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	for _, user := range s.users {
		if user.ID == id {
			return user, true
		}
	}
	return User{}, false
}

func (s *userStore) byEmail(email string) (User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	email = normalizeEmail(email)
	for _, user := range s.users {
		if user.Email == email {
			return user, true
		}
	}
	return User{}, false
}

var (
	errUserNotFound = errors.New("user not found")
	errEmailTaken   = errors.New("email already has an account")
	errLastOwner    = errors.New("the last owner cannot go")
)

// put inserts or replaces a user.
func (s *userStore) put(user User) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	user.Email = normalizeEmail(user.Email)
	user.Name = strings.TrimSpace(user.Name)
	user.Role = normalizeRole(user.Role)
	for _, other := range s.users {
		if other.Email == user.Email && other.ID != user.ID {
			return User{}, errEmailTaken
		}
	}
	if user.ID == "" {
		user.ID = "u" + randomHex(4)
		user.Created = timeNow().UTC()
		s.users = append(s.users, user)
		return user, s.saveLocked()
	}
	for index, existing := range s.users {
		if existing.ID == user.ID {
			if existing.Role == roleOwner && user.Role != roleOwner && s.ownersLocked() == 1 {
				return User{}, errLastOwner
			}
			s.users[index] = user
			return user, s.saveLocked()
		}
	}
	return User{}, errUserNotFound
}

func (s *userStore) ownersLocked() int {
	n := 0
	for _, user := range s.users {
		if user.Role == roleOwner {
			n++
		}
	}
	return n
}

func (s *userStore) remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	for index, user := range s.users {
		if user.ID != id {
			continue
		}
		if user.Role == roleOwner && s.ownersLocked() == 1 {
			return errLastOwner
		}
		s.users = append(s.users[:index], s.users[index+1:]...)
		kept := s.sessions[:0]
		for _, session := range s.sessions {
			if session.UserID != id {
				kept = append(kept, session)
			}
		}
		s.sessions = kept
		return s.saveLocked()
	}
	return errUserNotFound
}

// invite makes a one-use link for an email and role, replacing any earlier
// invite for the same address.
func (s *userStore) invite(email, role, by string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	email = normalizeEmail(email)
	kept := s.invites[:0]
	for _, invite := range s.invites {
		if invite.Email != email {
			kept = append(kept, invite)
		}
	}
	s.invites = kept
	token := newToken()
	s.invites = append(s.invites, Invite{TokenHash: hashToken(token), Email: email, Role: normalizeRole(role), Expires: timeNow().UTC().Add(inviteTTL), InvitedBy: by})
	return token, s.saveLocked()
}

func (s *userStore) inviteByToken(token string) (Invite, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	hash := hashToken(token)
	for _, invite := range s.invites {
		if invite.TokenHash == hash && invite.Expires.After(timeNow().UTC()) {
			return invite, true
		}
	}
	return Invite{}, false
}

func (s *userStore) consumeInvite(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	hash := hashToken(token)
	kept := s.invites[:0]
	for _, invite := range s.invites {
		if invite.TokenHash != hash {
			kept = append(kept, invite)
		}
	}
	s.invites = kept
	_ = s.saveLocked()
}

func (s *userStore) newSession(userID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	token := newToken()
	now := timeNow().UTC()
	s.sessions = append(s.sessions, Session{TokenHash: hashToken(token), UserID: userID, Created: now, Expires: now.Add(sessionTTL)})
	for index := range s.users {
		if s.users[index].ID == userID {
			s.users[index].LastLogin = now
		}
	}
	return token, s.saveLocked()
}

func (s *userStore) userForSession(token string) (User, bool) {
	if token == "" {
		return User{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	hash := hashToken(token)
	now := timeNow().UTC()
	for _, session := range s.sessions {
		if session.TokenHash == hash && session.Expires.After(now) {
			for _, user := range s.users {
				if user.ID == session.UserID {
					return user, true
				}
			}
		}
	}
	return User{}, false
}

func (s *userStore) dropSession(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	hash := hashToken(token)
	kept := s.sessions[:0]
	for _, session := range s.sessions {
		if session.TokenHash != hash {
			kept = append(kept, session)
		}
	}
	s.sessions = kept
	_ = s.saveLocked()
}

func (s *userStore) dropUserSessions(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	kept := s.sessions[:0]
	for _, session := range s.sessions {
		if session.UserID != userID {
			kept = append(kept, session)
		}
	}
	s.sessions = kept
	_ = s.saveLocked()
}

// ---------- passwords ----------

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), passwordCost)
	return string(hash), err
}

func checkPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func passwordProblem(password string) string {
	if len(password) < minPasswordLen {
		return "Use at least " + strconv.Itoa(minPasswordLen) + " characters. A few words you'll remember work well."
	}
	return ""
}

// ---------- one-time codes (TOTP, RFC 6238) ----------

func newTOTPSecret() string {
	raw := make([]byte, 20)
	_, _ = rand.Read(raw)
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw)
}

func totpCode(secret string, at time.Time) string {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(strings.ReplaceAll(secret, " ", "")))
	if err != nil {
		return ""
	}
	counter := uint64(at.Unix() / totpPeriod)
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], counter)
	mac := hmac.New(sha1.New, key)
	mac.Write(msg[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	code := (binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff) % 1000000
	out := strconv.Itoa(int(code))
	for len(out) < totpDigits {
		out = "0" + out
	}
	return out
}

// verifyTOTP accepts the current code and its neighbours, for clock drift.
func verifyTOTP(secret, code string, at time.Time) bool {
	code = strings.ReplaceAll(strings.TrimSpace(code), " ", "")
	if len(code) != totpDigits || secret == "" {
		return false
	}
	for _, step := range []int{0, -1, 1} {
		if hmac.Equal([]byte(totpCode(secret, at.Add(time.Duration(step*totpPeriod)*time.Second))), []byte(code)) {
			return true
		}
	}
	return false
}

// spacedSecret is the manual-entry form: groups of four.
func spacedSecret(secret string) string {
	var b strings.Builder
	for i, r := range secret {
		if i > 0 && i%4 == 0 {
			b.WriteByte(' ')
		}
		b.WriteRune(r)
	}
	return b.String()
}
