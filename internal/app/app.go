package app

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/openlist/openlist-cli/internal/spec"
)

const Version = "0.1.0"

type cliError struct {
	Code int
	Err  error
}

func (e cliError) Error() string { return e.Err.Error() }

type kvFlags []string

func (k *kvFlags) String() string { return strings.Join(*k, ",") }
func (k *kvFlags) Set(v string) error {
	*k = append(*k, v)
	return nil
}

type outputMode struct {
	JSON bool
	JQ   string
}

type listResult struct {
	Title      string               `json:"title"`
	Version    string               `json:"version"`
	Operations []spec.OperationSpec `json:"operations"`
}

type callResult struct {
	OperationID string              `json:"operation_id"`
	Method      string              `json:"method"`
	URL         string              `json:"url"`
	Status      int                 `json:"status"`
	Headers     map[string][]string `json:"headers"`
	Body        any                 `json:"body,omitempty"`
}

type routeResult struct {
	Kind string `json:"kind"`
	URL  string `json:"url"`
}

type fetchResult struct {
	URL        string              `json:"url"`
	Method     string              `json:"method"`
	Status     int                 `json:"status"`
	Headers    map[string][]string `json:"headers"`
	Bytes      int                 `json:"bytes"`
	Output     string              `json:"output,omitempty"`
	Body       string              `json:"body,omitempty"`
	BodyBase64 string              `json:"body_base64,omitempty"`
}

func Main(args []string, stdout, stderr io.Writer) int {
	if err := Run(args, stdout, stderr); err != nil {
		var ce cliError
		if errors.As(err, &ce) {
			fmt.Fprintln(stderr, ce.Err)
			return ce.Code
		}
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func Run(args []string, stdout, stderr io.Writer) error {
	doc, err := spec.Load()
	if err != nil {
		return cliError{Code: 1, Err: err}
	}
	if len(args) == 0 {
		printUsage(stdout)
		return nil
	}

	switch args[0] {
	case "help", "-h", "--help":
		printUsage(stdout)
		return nil
	case "version":
		_, err := fmt.Fprintln(stdout, Version)
		return err
	case "auth":
		return runAuth(doc, args[1:], stdout, stderr)
	case "config":
		return runConfig(args[1:], stdout)
	case "fs":
		return runFS(doc, args[1:], stdout)
	case "share":
		return runShare(doc, args[1:], stdout)
	case "list-ops":
		return runListOps(doc, args[1:], stdout)
	case "call":
		return runCall(doc, args[1:], stdout, stderr)
	case "route":
		return runRoute(args[1:], stdout)
	case "fetch":
		return runFetch(args[1:], stdout, stderr)
	default:
		return cliError{Code: 2, Err: fmt.Errorf("unknown command %q", args[0])}
	}
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, `openlist-cli is a friendly CLI for the OpenList API.

Friendly commands:
  openlist-cli auth login --username admin --password 'secret'
  openlist-cli auth token
  openlist-cli auth whoami
  openlist-cli fs ls /
  openlist-cli fs stat /Movies/file.mkv
  openlist-cli fs tree /Movies --depth 3
  openlist-cli fs search --parent / --keywords movie
  openlist-cli fs download-url /Movies/file.mkv
  openlist-cli share ls
  openlist-cli config show

Low-level commands:
  openlist-cli list-ops [--json|--plain] [--jq <expr>]
  openlist-cli call <operation-id> [flags]
  openlist-cli route <direct-url|proxy-url|archive-url|share-url> [flags]
  openlist-cli fetch --url <url-or-path> [flags]
  openlist-cli version

Common env:
  OPENLIST_BASE_URL   default base URL
  OPENLIST_TOKEN      API token sent as raw Authorization header
  OPENLIST_CLI_CONFIG config path override
`)
}

func runListOps(doc *spec.Document, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("list-ops", flag.ContinueOnError)
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
	result := listResult{Title: doc.Info.Title, Version: doc.Info.Version, Operations: spec.Operations(doc)}
	if mode.JSON {
		return renderJSON(stdout, result, mode.JQ)
	}
	for _, op := range result.Operations {
		auth := "public"
		if op.SecurityRequired {
			auth = "auth"
		}
		if _, err := fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\t%s\n", op.Tag, op.OperationID, op.Method, op.Path, auth); err != nil {
			return err
		}
	}
	return nil
}

func runCall(doc *spec.Document, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("call", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	baseURL := fs.String("base-url", defaultBaseURL(), "OpenList base URL")
	token := fs.String("token", defaultToken(), "API token sent as raw Authorization header")
	body := fs.String("body", "", "inline JSON request body")
	bodyFile := fs.String("body-file", "", "path to JSON body file")
	timeout := fs.Duration("timeout", 30*time.Second, "request timeout")
	insecure := fs.Bool("insecure", false, "skip TLS verification")
	jsonOut := fs.Bool("json", false, "emit JSON")
	plainOut := fs.Bool("plain", false, "emit plain text")
	jqExpr := fs.String("jq", "", "jq expression (JSON mode only)")
	var query kvFlags
	var headers kvFlags
	var pathParams kvFlags
	fs.Var(&query, "query", "query param key=value (repeatable)")
	fs.Var(&headers, "header", "extra header key=value (repeatable)")
	fs.Var(&pathParams, "path-param", "path param key=value (repeatable)")
	if err := fs.Parse(args); err != nil {
		return cliError{Code: 2, Err: err}
	}
	mode, err := resolveOutputMode(*jsonOut, *plainOut, *jqExpr)
	if err != nil {
		return cliError{Code: 2, Err: err}
	}
	if fs.NArg() < 1 {
		return cliError{Code: 2, Err: fmt.Errorf("missing operation-id")}
	}
	op, ok := spec.FindOperation(doc, fs.Arg(0))
	if !ok {
		return cliError{Code: 2, Err: fmt.Errorf("unknown operation-id %q", fs.Arg(0))}
	}
	requestBody, err := loadBody(*body, *bodyFile)
	if err != nil {
		return cliError{Code: 2, Err: err}
	}
	if op.RequestBodyRequired && requestBody == nil {
		fmt.Fprintf(stderr, "hint: %s accepts a request body; provide --body or --body-file\n", op.OperationID)
	}
	requestURL, err := buildOperationURL(strings.TrimSpace(*baseURL), op.Path, pathParams, query)
	if err != nil {
		return cliError{Code: 2, Err: err}
	}
	method := op.Method
	req, err := http.NewRequestWithContext(context.Background(), method, requestURL, bytes.NewReader(requestBody))
	if err != nil {
		return cliError{Code: 1, Err: err}
	}
	if len(requestBody) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	applyHeaders(req.Header, headers)
	if strings.TrimSpace(*token) != "" {
		req.Header.Set("Authorization", normalizeAuthToken(*token))
	} else if op.SecurityRequired {
		return cliError{Code: 2, Err: fmt.Errorf("operation %s requires auth; set --token or OPENLIST_TOKEN", op.OperationID)}
	}
	client := newHTTPClient(*timeout, *insecure)
	resp, err := client.Do(req)
	if err != nil {
		return cliError{Code: 1, Err: err}
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return cliError{Code: 1, Err: err}
	}
	result := callResult{
		OperationID: op.OperationID,
		Method:      op.Method,
		URL:         requestURL,
		Status:      resp.StatusCode,
		Headers:     resp.Header,
		Body:        decodeResponseBody(resp.Header.Get("Content-Type"), bodyBytes),
	}
	if mode.JSON {
		if err := renderJSON(stdout, result, mode.JQ); err != nil {
			return err
		}
	} else {
		if err := renderPlainCall(stdout, result); err != nil {
			return err
		}
	}
	if resp.StatusCode >= 400 {
		return cliError{Code: 1, Err: fmt.Errorf("request failed with status %d", resp.StatusCode)}
	}
	return nil
}

func runRoute(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return cliError{Code: 2, Err: fmt.Errorf("missing route subcommand")}
	}
	switch args[0] {
	case "direct-url":
		return runDirectRoute(args[1:], stdout, false)
	case "proxy-url":
		return runDirectRoute(args[1:], stdout, true)
	case "archive-url":
		return runArchiveRoute(args[1:], stdout)
	case "share-url":
		return runShareRoute(args[1:], stdout)
	default:
		return cliError{Code: 2, Err: fmt.Errorf("unknown route subcommand %q", args[0])}
	}
}

func runDirectRoute(args []string, stdout io.Writer, proxy bool) error {
	fs := flag.NewFlagSet("direct-url", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	baseURL := fs.String("base-url", defaultBaseURL(), "OpenList base URL")
	filePath := fs.String("path", "", "file path")
	sign := fs.String("sign", "", "download signature")
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
	if *filePath == "" {
		return cliError{Code: 2, Err: fmt.Errorf("--path is required")}
	}
	prefix := "/d"
	kind := "direct-url"
	if proxy {
		prefix = "/p"
		kind = "proxy-url"
	}
	routeURL, err := buildRouteURL(*baseURL, prefix, *filePath, map[string]string{"sign": *sign, "type": *linkType})
	if err != nil {
		return cliError{Code: 2, Err: err}
	}
	return emitRoute(stdout, mode, routeResult{Kind: kind, URL: routeURL})
}

func runArchiveRoute(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("archive-url", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	baseURL := fs.String("base-url", defaultBaseURL(), "OpenList base URL")
	modeFlag := fs.String("mode", "ad", "route mode: ad|ap|ae")
	archivePath := fs.String("archive-path", "", "archive file path")
	inner := fs.String("inner", "", "path inside archive")
	sign := fs.String("sign", "", "archive signature")
	pass := fs.String("pass", "", "archive password")
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
	if *archivePath == "" || *inner == "" {
		return cliError{Code: 2, Err: fmt.Errorf("--archive-path and --inner are required")}
	}
	route := "/ad"
	switch *modeFlag {
	case "ad", "ap", "ae":
		route = "/" + *modeFlag
	default:
		return cliError{Code: 2, Err: fmt.Errorf("invalid --mode %q", *modeFlag)}
	}
	routeURL, err := buildRouteURL(*baseURL, route, *archivePath, map[string]string{
		"sign":  *sign,
		"inner": *inner,
		"pass":  *pass,
		"type":  *linkType,
	})
	if err != nil {
		return cliError{Code: 2, Err: err}
	}
	return emitRoute(stdout, mode, routeResult{Kind: "archive-url", URL: routeURL})
}

func runShareRoute(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("share-url", flag.ContinueOnError)
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
	if *sharingID == "" {
		return cliError{Code: 2, Err: fmt.Errorf("--sharing-id is required")}
	}
	prefix := "/sd"
	kind := "share-url"
	if *archive {
		prefix = "/sad"
		kind = "share-archive-url"
	}
	joined := "/" + strings.TrimPrefix(*sharingID, "/")
	if strings.TrimSpace(*sharePath) != "" {
		joined += "/" + strings.TrimPrefix(*sharePath, "/")
	}
	routeURL, err := buildJoinedURL(*baseURL, prefix, joined)
	if err != nil {
		return cliError{Code: 2, Err: err}
	}
	return emitRoute(stdout, mode, routeResult{Kind: kind, URL: routeURL})
}

func runFetch(args []string, stdout, stderr io.Writer) error {
	_ = stderr
	fs := flag.NewFlagSet("fetch", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	baseURL := fs.String("base-url", defaultBaseURL(), "OpenList base URL")
	fetchURL := fs.String("url", "", "absolute URL or OpenList-relative path")
	output := fs.String("output", "", "write response body to file")
	head := fs.Bool("head", false, "use HEAD instead of GET")
	timeout := fs.Duration("timeout", 30*time.Second, "request timeout")
	insecure := fs.Bool("insecure", false, "skip TLS verification")
	jsonOut := fs.Bool("json", false, "emit JSON")
	plainOut := fs.Bool("plain", false, "emit plain text")
	jqExpr := fs.String("jq", "", "jq expression (JSON mode only)")
	var headers kvFlags
	fs.Var(&headers, "header", "extra header key=value (repeatable)")
	if err := fs.Parse(args); err != nil {
		return cliError{Code: 2, Err: err}
	}
	mode, err := resolveOutputMode(*jsonOut, *plainOut, *jqExpr)
	if err != nil {
		return cliError{Code: 2, Err: err}
	}
	if strings.TrimSpace(*fetchURL) == "" {
		return cliError{Code: 2, Err: fmt.Errorf("--url is required")}
	}
	finalURL, err := normalizeFetchURL(*baseURL, *fetchURL)
	if err != nil {
		return cliError{Code: 2, Err: err}
	}
	method := "GET"
	if *head {
		method = "HEAD"
	}
	req, err := http.NewRequestWithContext(context.Background(), method, finalURL, nil)
	if err != nil {
		return cliError{Code: 1, Err: err}
	}
	applyHeaders(req.Header, headers)
	resp, err := newHTTPClient(*timeout, *insecure).Do(req)
	if err != nil {
		return cliError{Code: 1, Err: err}
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return cliError{Code: 1, Err: err}
	}
	result := fetchResult{URL: finalURL, Method: method, Status: resp.StatusCode, Headers: resp.Header, Bytes: len(bodyBytes)}
	if *output != "" && !*head {
		if err := os.WriteFile(*output, bodyBytes, 0o644); err != nil {
			return cliError{Code: 1, Err: err}
		}
		result.Output = *output
	} else if !*head {
		if isTextContent(resp.Header.Get("Content-Type"), bodyBytes) {
			result.Body = string(bodyBytes)
		} else {
			result.BodyBase64 = base64.StdEncoding.EncodeToString(bodyBytes)
		}
	}
	if mode.JSON {
		if err := renderJSON(stdout, result, mode.JQ); err != nil {
			return err
		}
	} else {
		if result.Output != "" {
			_, err = fmt.Fprintln(stdout, result.Output)
		} else if result.Body != "" {
			_, err = io.WriteString(stdout, result.Body)
		} else {
			_, err = fmt.Fprintf(stdout, "status=%d bytes=%d\n", result.Status, result.Bytes)
		}
		if err != nil {
			return err
		}
	}
	if resp.StatusCode >= 400 {
		return cliError{Code: 1, Err: fmt.Errorf("fetch failed with status %d", resp.StatusCode)}
	}
	return nil
}

func resolveOutputMode(jsonOut, plainOut bool, jqExpr string) (outputMode, error) {
	if jsonOut && plainOut {
		return outputMode{}, fmt.Errorf("--json and --plain are mutually exclusive")
	}
	mode := outputMode{JSON: jsonOut, JQ: strings.TrimSpace(jqExpr)}
	if !jsonOut && !plainOut {
		mode.JSON = false
	}
	if mode.JQ != "" && !mode.JSON {
		return outputMode{}, fmt.Errorf("--jq requires --json")
	}
	return mode, nil
}

func renderJSON(stdout io.Writer, v any, jqExpr string) error {
	payload, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if jqExpr == "" {
		_, err = stdout.Write(append(payload, '\n'))
		return err
	}
	cmd := exec.Command("jq", jqExpr)
	cmd.Stdin = bytes.NewReader(payload)
	cmd.Stdout = stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return cliError{Code: 1, Err: fmt.Errorf("jq failed: %w", err)}
	}
	return nil
}

func renderPlainCall(stdout io.Writer, result callResult) error {
	bodyBytes, err := json.Marshal(result.Body)
	if err != nil {
		bodyBytes = []byte(fmt.Sprintf("%v", result.Body))
	}
	_, err = fmt.Fprintf(stdout, "operation_id=%s\nmethod=%s\nurl=%s\nstatus=%d\nbody=%s\n", result.OperationID, result.Method, result.URL, result.Status, string(bodyBytes))
	return err
}

func emitRoute(stdout io.Writer, mode outputMode, result routeResult) error {
	if mode.JSON {
		return renderJSON(stdout, result, mode.JQ)
	}
	_, err := fmt.Fprintln(stdout, result.URL)
	return err
}

func loadBody(body, bodyFile string) ([]byte, error) {
	body = strings.TrimSpace(body)
	bodyFile = strings.TrimSpace(bodyFile)
	if body != "" && bodyFile != "" {
		return nil, fmt.Errorf("use only one of --body or --body-file")
	}
	if body != "" {
		return []byte(body), nil
	}
	if bodyFile != "" {
		return os.ReadFile(bodyFile)
	}
	return nil, nil
}

func buildOperationURL(baseURL, rawPath string, pathParams, query kvFlags) (string, error) {
	u, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return "", err
	}
	decoded := rawPath
	escaped := rawPath
	for k, v := range parsePairs(pathParams) {
		decoded = strings.ReplaceAll(decoded, "{"+k+"}", v)
		escaped = strings.ReplaceAll(escaped, "{"+k+"}", url.PathEscape(v))
	}
	u.Path = joinURLPath(u.Path, decoded)
	u.RawPath = joinURLPath(u.EscapedPath(), escaped)
	q := u.Query()
	for k, v := range parsePairs(query) {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func buildRouteURL(baseURL, prefix, filePath string, query map[string]string) (string, error) {
	u, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return "", err
	}
	u.Path = joinURLPath(u.Path, prefix, filePath)
	u.RawPath = joinURLPath(u.EscapedPath(), prefix, encodePath(filePath))
	q := u.Query()
	keys := make([]string, 0, len(query))
	for k := range query {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if strings.TrimSpace(query[k]) != "" {
			q.Set(k, query[k])
		}
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func buildJoinedURL(baseURL, prefix, rest string) (string, error) {
	u, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return "", err
	}
	u.Path = joinURLPath(u.Path, prefix, rest)
	u.RawPath = joinURLPath(u.EscapedPath(), prefix, encodePath(rest))
	return u.String(), nil
}

func normalizeFetchURL(baseURL, raw string) (string, error) {
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw, nil
	}
	u, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return "", err
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	u.Path = joinURLPath(u.Path, parsed.Path)
	u.RawQuery = parsed.RawQuery
	return u.String(), nil
}

func decodeResponseBody(contentType string, body []byte) any {
	if len(body) == 0 {
		return nil
	}
	if isJSONContent(contentType) || json.Valid(body) {
		var decoded any
		if err := json.Unmarshal(body, &decoded); err == nil {
			return decoded
		}
	}
	return string(body)
}

func parsePairs(values kvFlags) map[string]string {
	result := make(map[string]string, len(values))
	for _, item := range values {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 {
			continue
		}
		result[strings.TrimSpace(parts[0])] = parts[1]
	}
	return result
}

func applyHeaders(h http.Header, values kvFlags) {
	for k, v := range parsePairs(values) {
		if strings.TrimSpace(k) == "" {
			continue
		}
		h.Set(k, v)
	}
}

func encodePath(p string) string {
	parts := strings.Split(strings.TrimPrefix(p, "/"), "/")
	encoded := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		encoded = append(encoded, url.PathEscape(part))
	}
	return strings.Join(encoded, "/")
}

func joinURLPath(parts ...string) string {
	clean := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" || p == "/" {
			continue
		}
		clean = append(clean, p)
	}
	joined := path.Join(clean...)
	if !strings.HasPrefix(joined, "/") {
		joined = "/" + joined
	}
	return joined
}

func newHTTPClient(timeout time.Duration, insecure bool) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: insecure} //nolint:gosec
	return &http.Client{Timeout: timeout, Transport: transport}
}

func normalizeAuthToken(token string) string {
	token = strings.TrimSpace(token)
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		return token[len("Bearer "):]
	}
	return token
}

func isJSONContent(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	return mediaType == "application/json" || strings.HasSuffix(mediaType, "+json")
}

func isTextContent(contentType string, body []byte) bool {
	if isJSONContent(contentType) {
		return true
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err == nil && strings.HasPrefix(mediaType, "text/") {
		return true
	}
	for _, b := range body {
		if b == 0 {
			return false
		}
	}
	return true
}
