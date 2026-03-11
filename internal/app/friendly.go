package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	pathpkg "path"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/openlist/openlist-cli/internal/spec"
)

type apiEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type lsItem struct {
	Name     string `json:"name"`
	Size     int64  `json:"size,omitempty"`
	IsDir    bool   `json:"is_dir"`
	Modified string `json:"modified,omitempty"`
	Sign     string `json:"sign,omitempty"`
	RawURL   string `json:"raw_url,omitempty"`
}

type loginResult struct {
	BaseURL string `json:"base_url"`
	Token   string `json:"token"`
	Saved   bool   `json:"saved"`
}

func runAuth(doc *spec.Document, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return cliError{Code: 2, Err: fmt.Errorf("missing auth subcommand")}
	}
	switch args[0] {
	case "login":
		return runAuthLogin(doc, args[1:], stdout, stderr)
	case "whoami":
		return runFriendlyWhoami(doc, args[1:], stdout)
	case "logout":
		return runAuthLogout(doc, args[1:], stdout)
	case "token":
		return runAuthToken(args[1:], stdout)
	default:
		return cliError{Code: 2, Err: fmt.Errorf("unknown auth subcommand %q", args[0])}
	}
}

func runConfig(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return cliError{Code: 2, Err: fmt.Errorf("missing config subcommand")}
	}
	switch args[0] {
	case "show":
		fs := flag.NewFlagSet("config show", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		jsonOut := fs.Bool("json", false, "emit JSON")
		plainOut := fs.Bool("plain", false, "emit plain text")
		jqExpr := fs.String("jq", "", "jq expression (JSON mode only)")
		if err := fs.Parse(args[1:]); err != nil {
			return cliError{Code: 2, Err: err}
		}
		mode, err := resolveOutputMode(*jsonOut, *plainOut, *jqExpr)
		if err != nil {
			return cliError{Code: 2, Err: err}
		}
		cfg, err := loadConfig()
		if err != nil {
			return cliError{Code: 1, Err: err}
		}
		masked := map[string]any{"base_url": cfg.BaseURL}
		if cfg.Token != "" {
			masked["token"] = maskToken(cfg.Token)
		}
		if mode.JSON {
			return renderJSON(stdout, masked, mode.JQ)
		}
		for _, key := range []string{"base_url", "token"} {
			if v, ok := masked[key]; ok {
				if _, err := fmt.Fprintf(stdout, "%s=%v\n", key, v); err != nil {
					return err
				}
			}
		}
		return nil
	case "set":
		fs := flag.NewFlagSet("config set", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		baseURL := fs.String("base-url", "", "OpenList base URL")
		token := fs.String("token", "", "API token sent as raw Authorization header")
		if err := fs.Parse(args[1:]); err != nil {
			return cliError{Code: 2, Err: err}
		}
		cfg, err := loadConfig()
		if err != nil {
			return cliError{Code: 1, Err: err}
		}
		if *baseURL != "" {
			cfg.BaseURL = strings.TrimSpace(*baseURL)
		}
		if *token != "" {
			cfg.Token = strings.TrimSpace(*token)
		}
		if cfg.BaseURL == "" && cfg.Token == "" {
			return cliError{Code: 2, Err: fmt.Errorf("nothing to set")}
		}
		if err := saveConfig(cfg); err != nil {
			return cliError{Code: 1, Err: err}
		}
		_, err = fmt.Fprintln(stdout, "ok")
		return err
	case "clear":
		cfg := config{}
		if err := saveConfig(cfg); err != nil {
			return cliError{Code: 1, Err: err}
		}
		_, err := fmt.Fprintln(stdout, "ok")
		return err
	default:
		return cliError{Code: 2, Err: fmt.Errorf("unknown config subcommand %q", args[0])}
	}
}

func runFS(doc *spec.Document, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return cliError{Code: 2, Err: fmt.Errorf("missing fs subcommand")}
	}
	switch args[0] {
	case "ls":
		return runFriendlyList(doc, args[1:], stdout)
	case "tree":
		return runFriendlyTree(doc, args[1:], stdout)
	case "stat":
		return runFriendlyStat(doc, args[1:], stdout)
	case "search":
		return runFriendlySearch(doc, args[1:], stdout)
	case "download-url":
		return runFriendlyDownloadURL(doc, args[1:], stdout)
	default:
		return cliError{Code: 2, Err: fmt.Errorf("unknown fs subcommand %q", args[0])}
	}
}

func runShare(doc *spec.Document, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return cliError{Code: 2, Err: fmt.Errorf("missing share subcommand")}
	}
	switch args[0] {
	case "ls":
		return runFriendlyShareList(doc, args[1:], stdout)
	case "url":
		return runFriendlyShareURL(args[1:], stdout)
	default:
		return cliError{Code: 2, Err: fmt.Errorf("unknown share subcommand %q", args[0])}
	}
}

func runAuthLogin(doc *spec.Document, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("auth login", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	baseURL := fs.String("base-url", defaultBaseURL(), "OpenList base URL")
	username := fs.String("username", "", "username")
	password := fs.String("password", "", "plain password")
	otpCode := fs.String("otp-code", "", "2FA code")
	save := fs.Bool("save", true, "save base URL and token to local config")
	jsonOut := fs.Bool("json", false, "emit JSON")
	plainOut := fs.Bool("plain", false, "emit plain text")
	jqExpr := fs.String("jq", "", "jq expression (JSON mode only)")
	if err := fs.Parse(args); err != nil {
		return cliError{Code: 2, Err: err}
	}
	mode, err := resolveOutputMode(*jsonOut, *plainOut, *jqExpr)
	if err != nil {
		return cliError{Code: 2, Err: err}
	}
	if *username == "" || *password == "" {
		return cliError{Code: 2, Err: fmt.Errorf("--username and --password are required")}
	}
	fmt.Fprintln(stderr, "logging in via /api/auth/login/hash")
	hash := sha256.Sum256([]byte(*password + "-https://github.com/alist-org/alist"))
	payload := map[string]any{
		"username": *username,
		"password": hex.EncodeToString(hash[:]),
	}
	if strings.TrimSpace(*otpCode) != "" {
		payload["otp_code"] = strings.TrimSpace(*otpCode)
	}
	body, _, err := doJSONOperation(doc, requestOptions{
		BaseURL:     *baseURL,
		OperationID: "loginHash",
		Body:        payload,
		Timeout:     30 * time.Second,
	})
	if err != nil {
		return err
	}
	token := strings.TrimSpace(getString(body, "token"))
	if token == "" {
		return cliError{Code: 1, Err: fmt.Errorf("login succeeded but token missing in response")}
	}
	if *save {
		cfg, _ := loadConfig()
		cfg.BaseURL = strings.TrimSpace(*baseURL)
		cfg.Token = token
		if err := saveConfig(cfg); err != nil {
			return cliError{Code: 1, Err: err}
		}
		fmt.Fprintln(stderr, "saved token to local config")
	}
	result := loginResult{BaseURL: strings.TrimSpace(*baseURL), Token: token, Saved: *save}
	if mode.JSON {
		return renderJSON(stdout, result, mode.JQ)
	}
	_, err = fmt.Fprintln(stdout, token)
	return err
}

func runFriendlyWhoami(doc *spec.Document, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("auth whoami", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	baseURL := fs.String("base-url", defaultBaseURL(), "OpenList base URL")
	token := fs.String("token", defaultToken(), "API token sent as raw Authorization header")
	jsonOut := fs.Bool("json", false, "emit JSON")
	plainOut := fs.Bool("plain", false, "emit plain text")
	jqExpr := fs.String("jq", "", "jq expression (JSON mode only)")
	if err := fs.Parse(args); err != nil {
		return cliError{Code: 2, Err: err}
	}
	mode, err := resolveOutputMode(*jsonOut, *plainOut, *jqExpr)
	if err != nil {
		return cliError{Code: 2, Err: err}
	}
	body, _, err := doJSONOperation(doc, requestOptions{BaseURL: *baseURL, Token: *token, OperationID: "currentUser", Timeout: 30 * time.Second})
	if err != nil {
		return err
	}
	if mode.JSON {
		return renderJSON(stdout, body, mode.JQ)
	}
	for _, key := range []string{"id", "username", "role"} {
		if v := getAny(body, key); v != nil {
			if _, err := fmt.Fprintf(stdout, "%s=%v\n", key, v); err != nil {
				return err
			}
		}
	}
	return nil
}

func runAuthToken(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("auth token", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOut := fs.Bool("json", false, "emit JSON")
	plainOut := fs.Bool("plain", false, "emit plain text")
	jqExpr := fs.String("jq", "", "jq expression (JSON mode only)")
	if err := fs.Parse(args); err != nil {
		return cliError{Code: 2, Err: err}
	}
	mode, err := resolveOutputMode(*jsonOut, *plainOut, *jqExpr)
	if err != nil {
		return cliError{Code: 2, Err: err}
	}
	token := normalizeAuthToken(defaultToken())
	if token == "" {
		return cliError{Code: 2, Err: fmt.Errorf("no token configured in --token/OPENLIST_TOKEN/config")}
	}
	if mode.JSON {
		return renderJSON(stdout, map[string]any{"token": token}, mode.JQ)
	}
	_, err = fmt.Fprintln(stdout, token)
	return err
}

func runAuthLogout(doc *spec.Document, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("auth logout", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	baseURL := fs.String("base-url", defaultBaseURL(), "OpenList base URL")
	token := fs.String("token", defaultToken(), "API token sent as raw Authorization header")
	server := fs.Bool("server", false, "also call remote logout API")
	if err := fs.Parse(args); err != nil {
		return cliError{Code: 2, Err: err}
	}
	cfg, _ := loadConfig()
	cfg.Token = ""
	if err := saveConfig(cfg); err != nil {
		return cliError{Code: 1, Err: err}
	}
	if *server && strings.TrimSpace(*token) != "" {
		_, _, err := doJSONOperation(doc, requestOptions{BaseURL: *baseURL, Token: *token, OperationID: "logout", Timeout: 30 * time.Second})
		if err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(stdout, "ok")
	return err
}

func runFriendlyList(doc *spec.Document, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("fs ls", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	baseURL := fs.String("base-url", defaultBaseURL(), "OpenList base URL")
	token := fs.String("token", defaultToken(), "API token sent as raw Authorization header")
	listPath := fs.String("path", "/", "directory path")
	password := fs.String("password", "", "meta password")
	page := fs.Int("page", 1, "page number")
	perPage := fs.Int("per-page", 0, "page size, 0=all")
	refresh := fs.Bool("refresh", false, "refresh directory cache")
	jsonOut := fs.Bool("json", false, "emit JSON")
	plainOut := fs.Bool("plain", false, "emit plain text")
	jqExpr := fs.String("jq", "", "jq expression (JSON mode only)")
	if err := fs.Parse(args); err != nil {
		return cliError{Code: 2, Err: err}
	}
	mode, err := resolveOutputMode(*jsonOut, *plainOut, *jqExpr)
	if err != nil {
		return cliError{Code: 2, Err: err}
	}
	pathValue, err := resolveOptionalPathArg(fs, *listPath, "/")
	if err != nil {
		return cliError{Code: 2, Err: err}
	}
	body, _, err := doJSONOperation(doc, requestOptions{
		BaseURL:     *baseURL,
		Token:       *token,
		OperationID: "fsList",
		Body:        map[string]any{"path": pathValue, "password": *password, "page": *page, "per_page": *perPage, "refresh": *refresh},
		Timeout:     30 * time.Second,
	})
	if err != nil {
		return err
	}
	if mode.JSON {
		return renderJSON(stdout, body, mode.JQ)
	}
	content, _ := body["content"].([]any)
	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "TYPE\tSIZE\tMODIFIED\tNAME\tPATH"); err != nil {
		return err
	}
	for _, item := range content {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		kind := "file"
		if getBool(m, "is_dir") {
			kind = "dir"
		}
		fullPath := joinRemotePath(pathValue, getString(m, "name"))
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", kind, formatSizeCell(m, kind == "dir"), getString(m, "modified"), getString(m, "name"), strconv.Quote(fullPath)); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func runFriendlyTree(doc *spec.Document, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("fs tree", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	baseURL := fs.String("base-url", defaultBaseURL(), "OpenList base URL")
	token := fs.String("token", defaultToken(), "API token sent as raw Authorization header")
	rootPath := fs.String("path", "/", "directory path")
	password := fs.String("password", "", "meta password")
	depth := fs.Int("depth", 3, "tree depth")
	jsonOut := fs.Bool("json", false, "emit JSON")
	plainOut := fs.Bool("plain", false, "emit plain text")
	jqExpr := fs.String("jq", "", "jq expression (JSON mode only)")
	if err := fs.Parse(args); err != nil {
		return cliError{Code: 2, Err: err}
	}
	mode, err := resolveOutputMode(*jsonOut, *plainOut, *jqExpr)
	if err != nil {
		return cliError{Code: 2, Err: err}
	}
	if *depth < 1 {
		return cliError{Code: 2, Err: fmt.Errorf("--depth must be >= 1")}
	}
	pathValue, err := resolveOptionalPathArg(fs, *rootPath, "/")
	if err != nil {
		return cliError{Code: 2, Err: err}
	}
	root, err := buildTreeNode(doc, requestOptions{
		BaseURL:     *baseURL,
		Token:       *token,
		OperationID: "fsList",
		Timeout:     30 * time.Second,
	}, pathValue, *password, *depth)
	if err != nil {
		return err
	}
	if mode.JSON {
		return renderJSON(stdout, root, mode.JQ)
	}
	if _, err := fmt.Fprintln(stdout, root.Path); err != nil {
		return err
	}
	for i, child := range root.Children {
		if err := writeTreeNode(stdout, child, "", i == len(root.Children)-1); err != nil {
			return err
		}
	}
	return nil
}

func runFriendlyStat(doc *spec.Document, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("fs stat", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	baseURL := fs.String("base-url", defaultBaseURL(), "OpenList base URL")
	token := fs.String("token", defaultToken(), "API token sent as raw Authorization header")
	filePath := fs.String("path", "", "file or directory path")
	password := fs.String("password", "", "meta password")
	jsonOut := fs.Bool("json", false, "emit JSON")
	plainOut := fs.Bool("plain", false, "emit plain text")
	jqExpr := fs.String("jq", "", "jq expression (JSON mode only)")
	if err := fs.Parse(args); err != nil {
		return cliError{Code: 2, Err: err}
	}
	mode, err := resolveOutputMode(*jsonOut, *plainOut, *jqExpr)
	if err != nil {
		return cliError{Code: 2, Err: err}
	}
	pathValue, err := resolveRequiredPathArg(fs, *filePath)
	if err != nil {
		return cliError{Code: 2, Err: err}
	}
	body, _, err := doJSONOperation(doc, requestOptions{BaseURL: *baseURL, Token: *token, OperationID: "fsGet", Body: map[string]any{"path": pathValue, "password": *password}, Timeout: 30 * time.Second})
	if err != nil {
		return err
	}
	if mode.JSON {
		return renderJSON(stdout, body, mode.JQ)
	}
	for _, key := range []string{"name", "size", "is_dir", "modified", "raw_url", "sign"} {
		if v := getAny(body, key); v != nil {
			if _, err := fmt.Fprintf(stdout, "%s=%v\n", key, v); err != nil {
				return err
			}
		}
	}
	return nil
}

func runFriendlySearch(doc *spec.Document, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("fs search", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	baseURL := fs.String("base-url", defaultBaseURL(), "OpenList base URL")
	token := fs.String("token", defaultToken(), "API token sent as raw Authorization header")
	parent := fs.String("parent", "/", "parent path")
	keywords := fs.String("keywords", "", "search keywords")
	scope := fs.Int("scope", 0, "0=all,1=folders,2=files")
	page := fs.Int("page", 1, "page number")
	perPage := fs.Int("per-page", 0, "page size, 0=all")
	password := fs.String("password", "", "meta password")
	jsonOut := fs.Bool("json", false, "emit JSON")
	plainOut := fs.Bool("plain", false, "emit plain text")
	jqExpr := fs.String("jq", "", "jq expression (JSON mode only)")
	if err := fs.Parse(args); err != nil {
		return cliError{Code: 2, Err: err}
	}
	mode, err := resolveOutputMode(*jsonOut, *plainOut, *jqExpr)
	if err != nil {
		return cliError{Code: 2, Err: err}
	}
	if strings.TrimSpace(*keywords) == "" {
		return cliError{Code: 2, Err: fmt.Errorf("--keywords is required")}
	}
	body, _, err := doJSONOperation(doc, requestOptions{BaseURL: *baseURL, Token: *token, OperationID: "fsSearch", Body: map[string]any{"parent": *parent, "keywords": *keywords, "scope": *scope, "page": *page, "per_page": *perPage, "password": *password}, Timeout: 30 * time.Second})
	if err != nil {
		return err
	}
	if mode.JSON {
		return renderJSON(stdout, body, mode.JQ)
	}
	content, _ := body["content"].([]any)
	for _, item := range content {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		kind := "file"
		if getBool(m, "is_dir") {
			kind = "dir"
		}
		if _, err := fmt.Fprintf(stdout, "%s\t%s/%s\t%v\n", kind, getString(m, "parent"), getString(m, "name"), getAny(m, "size")); err != nil {
			return err
		}
	}
	return nil
}

func runFriendlyDownloadURL(doc *spec.Document, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("fs download-url", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	baseURL := fs.String("base-url", defaultBaseURL(), "OpenList base URL")
	token := fs.String("token", defaultToken(), "API token sent as raw Authorization header")
	filePath := fs.String("path", "", "file path")
	password := fs.String("password", "", "meta password")
	proxy := fs.Bool("proxy", false, "build /p URL instead of /d")
	rawURL := fs.Bool("raw-url", false, "prefer storage raw_url from /api/fs/get")
	linkType := fs.String("type", "", "storage link type hint")
	jsonOut := fs.Bool("json", false, "emit JSON")
	plainOut := fs.Bool("plain", false, "emit plain text")
	jqExpr := fs.String("jq", "", "jq expression (JSON mode only)")
	if err := fs.Parse(args); err != nil {
		return cliError{Code: 2, Err: err}
	}
	mode, err := resolveOutputMode(*jsonOut, *plainOut, *jqExpr)
	if err != nil {
		return cliError{Code: 2, Err: err}
	}
	pathValue, err := resolveRequiredPathArg(fs, *filePath)
	if err != nil {
		return cliError{Code: 2, Err: err}
	}
	body, _, err := doJSONOperation(doc, requestOptions{BaseURL: *baseURL, Token: *token, OperationID: "fsGet", Body: map[string]any{"path": pathValue, "password": *password}, Timeout: 30 * time.Second})
	if err != nil {
		return err
	}
	result := map[string]any{"path": pathValue, "sign": getString(body, "sign"), "raw_url": getString(body, "raw_url")}
	if *rawURL && getString(body, "raw_url") != "" {
		result["url"] = getString(body, "raw_url")
		result["kind"] = "raw-url"
		if mode.JSON {
			return renderJSON(stdout, result, mode.JQ)
		}
		_, err = fmt.Fprintln(stdout, result["url"])
		return err
	}
	prefix := "/d"
	kind := "direct-url"
	if *proxy {
		prefix = "/p"
		kind = "proxy-url"
	}
	routeURL, err := buildRouteURL(*baseURL, prefix, pathValue, map[string]string{"sign": getString(body, "sign"), "type": *linkType})
	if err != nil {
		return cliError{Code: 2, Err: err}
	}
	result["kind"] = kind
	result["url"] = routeURL
	if mode.JSON {
		return renderJSON(stdout, result, mode.JQ)
	}
	_, err = fmt.Fprintln(stdout, routeURL)
	return err
}

func runFriendlyShareURL(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("share url", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	baseURL := fs.String("base-url", defaultBaseURL(), "OpenList base URL")
	sharingID := fs.String("sharing-id", "", "share ID")
	sharePath := fs.String("path", "", "path inside share")
	archive := fs.Bool("archive", false, "use /sad route")
	jsonOut := fs.Bool("json", false, "emit JSON")
	plainOut := fs.Bool("plain", false, "emit plain text")
	jqExpr := fs.String("jq", "", "jq expression (JSON mode only)")
	if err := fs.Parse(args); err != nil {
		return cliError{Code: 2, Err: err}
	}
	mode, err := resolveOutputMode(*jsonOut, *plainOut, *jqExpr)
	if err != nil {
		return cliError{Code: 2, Err: err}
	}
	if fs.NArg() > 2 {
		return cliError{Code: 2, Err: fmt.Errorf("too many positional arguments")}
	}
	sharingIDValue := strings.TrimSpace(*sharingID)
	if sharingIDValue == "" && fs.NArg() >= 1 {
		sharingIDValue = strings.TrimSpace(fs.Arg(0))
	}
	sharePathValue := strings.TrimSpace(*sharePath)
	if sharePathValue == "" && fs.NArg() >= 2 {
		sharePathValue = strings.TrimSpace(fs.Arg(1))
	}
	if sharingIDValue == "" {
		return cliError{Code: 2, Err: fmt.Errorf("missing share id")}
	}
	prefix := "/sd"
	kind := "share-url"
	if *archive {
		prefix = "/sad"
		kind = "share-archive-url"
	}
	joined := "/" + strings.TrimPrefix(sharingIDValue, "/")
	if sharePathValue != "" {
		joined += "/" + strings.TrimPrefix(sharePathValue, "/")
	}
	routeURL, err := buildJoinedURL(*baseURL, prefix, joined)
	if err != nil {
		return cliError{Code: 2, Err: err}
	}
	return emitRoute(stdout, mode, routeResult{Kind: kind, URL: routeURL})
}

func runFriendlyShareList(doc *spec.Document, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("share ls", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	baseURL := fs.String("base-url", defaultBaseURL(), "OpenList base URL")
	token := fs.String("token", defaultToken(), "API token sent as raw Authorization header")
	page := fs.Int("page", 1, "page number")
	perPage := fs.Int("per-page", 0, "page size, 0=all")
	jsonOut := fs.Bool("json", false, "emit JSON")
	plainOut := fs.Bool("plain", false, "emit plain text")
	jqExpr := fs.String("jq", "", "jq expression (JSON mode only)")
	if err := fs.Parse(args); err != nil {
		return cliError{Code: 2, Err: err}
	}
	mode, err := resolveOutputMode(*jsonOut, *plainOut, *jqExpr)
	if err != nil {
		return cliError{Code: 2, Err: err}
	}
	body, _, err := doJSONOperation(doc, requestOptions{BaseURL: *baseURL, Token: *token, OperationID: "listSharings", Body: map[string]any{"page": *page, "per_page": *perPage}, Timeout: 30 * time.Second})
	if err != nil {
		return err
	}
	if mode.JSON {
		return renderJSON(stdout, body, mode.JQ)
	}
	content, _ := body["content"].([]any)
	for _, item := range content {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id := firstNonEmpty(getString(m, "id"), getString(m, "share_id"))
		if _, err := fmt.Fprintf(stdout, "%s\t%s\t%v\n", id, getString(m, "pwd"), getAny(m, "files")); err != nil {
			return err
		}
	}
	return nil
}

type treeNode struct {
	Name     string     `json:"name"`
	Path     string     `json:"path"`
	IsDir    bool       `json:"is_dir"`
	Size     int64      `json:"size,omitempty"`
	Modified string     `json:"modified,omitempty"`
	Sign     string     `json:"sign,omitempty"`
	Children []treeNode `json:"children,omitempty"`
}

type requestOptions struct {
	BaseURL     string
	Token       string
	OperationID string
	Body        any
	Timeout     time.Duration
}

func doJSONOperation(doc *spec.Document, opts requestOptions) (map[string]any, apiEnvelope, error) {
	op, ok := spec.FindOperation(doc, opts.OperationID)
	if !ok {
		return nil, apiEnvelope{}, cliError{Code: 2, Err: fmt.Errorf("unknown operation-id %q", opts.OperationID)}
	}
	var bodyBytes []byte
	var err error
	if opts.Body != nil {
		bodyBytes, err = json.Marshal(opts.Body)
		if err != nil {
			return nil, apiEnvelope{}, cliError{Code: 2, Err: err}
		}
	}
	requestURL, err := buildOperationURL(strings.TrimSpace(opts.BaseURL), op.Path, nil, nil)
	if err != nil {
		return nil, apiEnvelope{}, cliError{Code: 2, Err: err}
	}
	req, err := http.NewRequestWithContext(context.Background(), op.Method, requestURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, apiEnvelope{}, cliError{Code: 1, Err: err}
	}
	if len(bodyBytes) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	if strings.TrimSpace(opts.Token) != "" {
		req.Header.Set("Authorization", normalizeAuthToken(opts.Token))
	} else if op.SecurityRequired {
		return nil, apiEnvelope{}, cliError{Code: 2, Err: fmt.Errorf("operation %s requires auth; run auth login or set OPENLIST_TOKEN", op.OperationID)}
	}
	resp, err := newHTTPClient(opts.Timeout, false).Do(req)
	if err != nil {
		return nil, apiEnvelope{}, cliError{Code: 1, Err: err}
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, apiEnvelope{}, cliError{Code: 1, Err: err}
	}
	var env apiEnvelope
	if err := json.Unmarshal(payload, &env); err != nil {
		return nil, apiEnvelope{}, cliError{Code: 1, Err: fmt.Errorf("decode API response: %w", err)}
	}
	var data map[string]any
	if len(env.Data) > 0 && string(env.Data) != "null" {
		if err := json.Unmarshal(env.Data, &data); err != nil {
			var scalar any
			if err := json.Unmarshal(env.Data, &scalar); err == nil {
				data = map[string]any{"value": scalar}
			} else {
				return nil, apiEnvelope{}, cliError{Code: 1, Err: fmt.Errorf("decode response data: %w", err)}
			}
		}
	} else {
		data = map[string]any{}
	}
	if resp.StatusCode >= 400 || env.Code >= 300 {
		msg := strings.TrimSpace(env.Message)
		if msg == "" {
			msg = fmt.Sprintf("request failed with status %d", resp.StatusCode)
		}
		return data, env, cliError{Code: 1, Err: fmt.Errorf("%s", msg)}
	}
	return data, env, nil
}

func getString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func getAny(m map[string]any, key string) any {
	if m == nil {
		return nil
	}
	return m[key]
}

func getBool(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	v, ok := m[key].(bool)
	return ok && v
}

func buildTreeNode(doc *spec.Document, req requestOptions, currentPath, password string, depth int) (treeNode, error) {
	body, _, err := doJSONOperation(doc, requestOptions{
		BaseURL:     req.BaseURL,
		Token:       req.Token,
		OperationID: "fsList",
		Body:        map[string]any{"path": currentPath, "password": password, "page": 1, "per_page": 0, "refresh": false},
		Timeout:     req.Timeout,
	})
	if err != nil {
		return treeNode{}, err
	}
	node := treeNode{
		Name:     pathBase(currentPath),
		Path:     currentPath,
		IsDir:    true,
		Modified: getString(body, "modified"),
	}
	if currentPath == "/" {
		node.Name = "/"
	}
	if depth <= 1 {
		return node, nil
	}
	content, _ := body["content"].([]any)
	children := make([]treeNode, 0, len(content))
	for _, item := range content {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		child := treeNode{
			Name:     getString(m, "name"),
			Path:     joinRemotePath(currentPath, getString(m, "name")),
			IsDir:    getBool(m, "is_dir"),
			Size:     getInt64(m, "size"),
			Modified: getString(m, "modified"),
			Sign:     getString(m, "sign"),
		}
		if child.IsDir && depth > 1 {
			subtree, err := buildTreeNode(doc, req, child.Path, password, depth-1)
			if err != nil {
				return treeNode{}, err
			}
			child.Children = subtree.Children
		}
		children = append(children, child)
	}
	node.Children = children
	return node, nil
}

func writeTreeNode(w io.Writer, node treeNode, prefix string, last bool) error {
	branch := "├── "
	nextPrefix := prefix + "│   "
	if last {
		branch = "└── "
		nextPrefix = prefix + "    "
	}
	label := node.Name
	if node.IsDir {
		label += "/"
	}
	if _, err := fmt.Fprintf(w, "%s%s%s\n", prefix, branch, label); err != nil {
		return err
	}
	for i, child := range node.Children {
		if err := writeTreeNode(w, child, nextPrefix, i == len(node.Children)-1); err != nil {
			return err
		}
	}
	return nil
}

func pathBase(p string) string {
	clean := strings.TrimSpace(p)
	if clean == "" || clean == "/" {
		return "/"
	}
	return pathpkg.Base(clean)
}

func resolveOptionalPathArg(fs *flag.FlagSet, flagValue, fallback string) (string, error) {
	if fs.NArg() > 1 {
		return "", fmt.Errorf("too many positional arguments")
	}
	pathValue := strings.TrimSpace(flagValue)
	if pathValue == "" || pathValue == fallback {
		if fs.NArg() == 1 {
			pathValue = strings.TrimSpace(fs.Arg(0))
		}
	}
	if pathValue == "" || pathValue == "." {
		return fallback, nil
	}
	return pathValue, nil
}

func resolveRequiredPathArg(fs *flag.FlagSet, flagValue string) (string, error) {
	if fs.NArg() > 1 {
		return "", fmt.Errorf("too many positional arguments")
	}
	pathValue := strings.TrimSpace(flagValue)
	if pathValue == "" && fs.NArg() == 1 {
		pathValue = strings.TrimSpace(fs.Arg(0))
	}
	if pathValue == "" {
		return "", fmt.Errorf("missing path")
	}
	if pathValue == "." {
		return "/", nil
	}
	return pathValue, nil
}

func joinRemotePath(parent, name string) string {
	if strings.TrimSpace(parent) == "" || parent == "." {
		parent = "/"
	}
	return pathpkg.Join(parent, name)
}

func formatSizeCell(m map[string]any, isDir bool) string {
	if isDir {
		return "-"
	}
	size := getInt64(m, "size")
	return fmt.Sprintf("%d (%s)", size, humanSize(size))
}

func getInt64(m map[string]any, key string) int64 {
	if m == nil {
		return 0
	}
	switch v := m[key].(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case json.Number:
		n, _ := v.Int64()
		return n
	default:
		return 0
	}
}

func humanSize(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	units := []string{"B", "KiB", "MiB", "GiB", "TiB"}
	size := float64(n)
	unit := 0
	for size >= 1024 && unit < len(units)-1 {
		size /= 1024
		unit++
	}
	return fmt.Sprintf("%.2f %s", size, units[unit])
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func maskToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "****" + token[len(token)-4:]
}

func friendlyCommands() []string {
	commands := []string{
		"auth login",
		"auth whoami",
		"auth logout",
		"config show",
		"config set",
		"config clear",
		"fs ls",
		"fs stat",
		"fs tree",
		"fs search",
		"fs download-url",
		"share ls",
		"share url",
	}
	sort.Strings(commands)
	return commands
}
