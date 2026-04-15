package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/headercat/erdn-lang/internal/parser"
	"github.com/headercat/erdn-lang/internal/render"
	"github.com/headercat/erdn-lang/internal/semantic"
	"github.com/headercat/erdn-lang/internal/sqlimport"
)

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func main() {
	br := bufio.NewReader(os.Stdin)
	bw := bufio.NewWriter(os.Stdout)
	for {
		msg, err := readMessage(br)
		if err != nil {
			if err == io.EOF {
				return
			}
			_ = writeJSON(bw, rpcResponse{
				JSONRPC: "2.0",
				Error:   &rpcError{Code: -32700, Message: "parse error"},
			})
			continue
		}

		var req rpcRequest
		if err := json.Unmarshal(msg, &req); err != nil {
			_ = writeJSON(bw, rpcResponse{
				JSONRPC: "2.0",
				Error:   &rpcError{Code: -32600, Message: "invalid request"},
			})
			continue
		}

		// Notifications do not require responses.
		if len(req.ID) == 0 {
			continue
		}
		id := decodeID(req.ID)

		switch req.Method {
		case "initialize":
			_ = writeJSON(bw, rpcResponse{
				JSONRPC: "2.0",
				ID:      id,
				Result: map[string]interface{}{
					"protocolVersion": "2024-11-05",
					"capabilities": map[string]interface{}{
						"tools": map[string]interface{}{},
					},
					"serverInfo": map[string]string{
						"name":    "erdn-lang-mcp",
						"version": "0.1.0",
					},
				},
			})
		case "tools/list":
			_ = writeJSON(bw, rpcResponse{
				JSONRPC: "2.0",
				ID:      id,
				Result: map[string]interface{}{
					"tools": []map[string]interface{}{
						{
							"name":        "convert_sql_to_erdn",
							"description": "Convert SQL CREATE TABLE/FK schema text to ERDN source code.",
							"inputSchema": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"sql": map[string]string{
										"type":        "string",
										"description": "SQL DDL text containing CREATE TABLE statements.",
									},
								},
								"required": []string{"sql"},
							},
						},
						{
							"name":        "generate_svg_from_erdn",
							"description": "Validate ERDN source and generate an SVG diagram.",
							"inputSchema": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"erdn": map[string]string{
										"type":        "string",
										"description": "ERDN schema source text.",
									},
								},
								"required": []string{"erdn"},
							},
						},
					},
				},
			})
		case "tools/call":
			res, rerr := handleToolCall(req.Params)
			if rerr != nil {
				_ = writeJSON(bw, rpcResponse{
					JSONRPC: "2.0",
					ID:      id,
					Result: map[string]interface{}{
						"content": []map[string]string{
							{"type": "text", "text": rerr.Error()},
						},
						"isError": true,
					},
				})
				continue
			}
			_ = writeJSON(bw, rpcResponse{
				JSONRPC: "2.0",
				ID:      id,
				Result:  res,
			})
		default:
			_ = writeJSON(bw, rpcResponse{
				JSONRPC: "2.0",
				ID:      id,
				Error:   &rpcError{Code: -32601, Message: "method not found"},
			})
		}
	}
}

func handleToolCall(params json.RawMessage) (map[string]interface{}, error) {
	var call struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	if err := json.Unmarshal(params, &call); err != nil {
		return nil, fmt.Errorf("invalid tools/call params: %w", err)
	}

	switch call.Name {
	case "convert_sql_to_erdn":
		sql, _ := call.Arguments["sql"].(string)
		if strings.TrimSpace(sql) == "" {
			return nil, fmt.Errorf("sql is required")
		}
		prog, err := sqlimport.ParseDDL(sql)
		if err != nil {
			return nil, err
		}
		erdn := sqlimport.ToERDN(prog)
		return map[string]interface{}{
			"content": []map[string]string{
				{"type": "text", "text": erdn},
			},
		}, nil
	case "generate_svg_from_erdn":
		src, _ := call.Arguments["erdn"].(string)
		if strings.TrimSpace(src) == "" {
			return nil, fmt.Errorf("erdn is required")
		}
		prog, err := parser.ParseString(src)
		if err != nil {
			return nil, err
		}
		if errs := semantic.Validate(prog); len(errs) > 0 {
			var lines []string
			for _, e := range errs {
				lines = append(lines, e.Error())
			}
			return nil, fmt.Errorf(strings.Join(lines, "\n"))
		}
		svg := render.GenerateSVG(prog)
		return map[string]interface{}{
			"content": []map[string]string{
				{"type": "text", "text": svg},
			},
		}, nil
	default:
		return nil, fmt.Errorf("unknown tool: %s", call.Name)
	}
}

func decodeID(raw json.RawMessage) interface{} {
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	return v
}

func readMessage(r *bufio.Reader) ([]byte, error) {
	headers := map[string]string{}
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid header line")
		}
		headers[strings.ToLower(strings.TrimSpace(parts[0]))] = strings.TrimSpace(parts[1])
	}
	cl, ok := headers["content-length"]
	if !ok {
		return nil, fmt.Errorf("missing content-length header")
	}
	n, err := strconv.Atoi(cl)
	if err != nil || n < 0 {
		return nil, fmt.Errorf("invalid content-length")
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func writeJSON(w *bufio.Writer, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	var frame bytes.Buffer
	fmt.Fprintf(&frame, "Content-Length: %d\r\n\r\n", len(body))
	frame.Write(body)
	if _, err := w.Write(frame.Bytes()); err != nil {
		return err
	}
	return w.Flush()
}
