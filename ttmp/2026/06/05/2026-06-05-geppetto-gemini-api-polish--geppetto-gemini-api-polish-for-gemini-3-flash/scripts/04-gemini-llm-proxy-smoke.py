#!/usr/bin/env python3
"""Run Gemini-backed llm-proxy OpenAI-compatible smoke tests.

The script starts llm-proxy against the local Geppetto/Pinocchio profile registry,
then exercises /v1/models, /v1/completions, and /v1/chat/completions. Provider
credentials stay in the profile registry; this script does not read API-key env vars.
"""

from __future__ import annotations

import argparse
import json
import os
import signal
import socket
import subprocess
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any

TICKET = "2026-06-05-geppetto-gemini-api-polish--geppetto-gemini-api-polish-for-gemini-3-flash"
DEFAULT_ARTIFACTS = Path("ttmp/2026/06/05") / TICKET / "scripts" / "artifacts"
DEFAULT_PROFILES = Path.home() / ".config" / "pinocchio" / "profiles.yaml"
DEFAULT_LLM_PROXY_REPO = Path(__file__).resolve().parents[7] / "llm-proxy"


def post_json(url: str, payload: dict[str, Any], timeout: int = 180) -> tuple[int, dict[str, str], bytes]:
    data = json.dumps(payload).encode("utf-8")
    req = urllib.request.Request(url, data=data, headers={"Content-Type": "application/json"}, method="POST")
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            return resp.status, dict(resp.headers.items()), resp.read()
    except urllib.error.HTTPError as e:
        return e.code, dict(e.headers.items()), e.read()


def get_json(url: str, timeout: int = 30) -> tuple[int, dict[str, str], bytes]:
    try:
        with urllib.request.urlopen(url, timeout=timeout) as resp:
            return resp.status, dict(resp.headers.items()), resp.read()
    except urllib.error.HTTPError as e:
        return e.code, dict(e.headers.items()), e.read()


def write_json(path: Path, value: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, indent=2, sort_keys=True) + "\n")


def write_bytes(path: Path, value: bytes) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_bytes(value)


def decode_json_bytes(raw: bytes) -> Any:
    try:
        return json.loads(raw.decode("utf-8"))
    except Exception as e:  # noqa: BLE001 - artifact helper should preserve decode failures
        return {"decode_error": str(e), "raw": raw.decode("utf-8", errors="replace")}


def allocate_listen_addr(listen: str) -> str:
    if not listen.endswith(":0"):
        return listen
    host = listen.rsplit(":", 1)[0]
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind((host, 0))
        port = sock.getsockname()[1]
    return f"{host}:{port}"


def wait_for_health(base_url: str, proc: subprocess.Popen[bytes], timeout_s: float = 60) -> None:
    deadline = time.time() + timeout_s
    last_error = ""
    while time.time() < deadline:
        if proc.poll() is not None:
            raise RuntimeError(f"llm-proxy exited early with status {proc.returncode}")
        try:
            status, _, raw = get_json(f"{base_url}/healthz", timeout=2)
            if status == 200 and b"ok" in raw:
                return
            last_error = f"status={status} body={raw[:200]!r}"
        except Exception as e:  # noqa: BLE001
            last_error = str(e)
        time.sleep(0.25)
    raise TimeoutError(f"llm-proxy did not become healthy: {last_error}")


def stop_process(proc: subprocess.Popen[bytes]) -> None:
    if proc.poll() is not None:
        return
    proc.send_signal(signal.SIGTERM)
    try:
        proc.wait(timeout=5)
    except subprocess.TimeoutExpired:
        proc.kill()
        proc.wait(timeout=5)


def summarize_chat_response(body: Any) -> dict[str, Any]:
    choice = (body.get("choices") or [{}])[0] if isinstance(body, dict) else {}
    msg = choice.get("message") or {}
    return {
        "id": body.get("id") if isinstance(body, dict) else None,
        "finish_reason": choice.get("finish_reason"),
        "content": msg.get("content"),
        "tool_calls": msg.get("tool_calls") or [],
        "usage": body.get("usage") if isinstance(body, dict) else None,
    }


def run_case(base_url: str, artifacts: Path, name: str, endpoint: str, payload: dict[str, Any]) -> dict[str, Any]:
    request_path = artifacts / f"llm-proxy-gemini-{name}-request.json"
    response_path = artifacts / f"llm-proxy-gemini-{name}-response.json"
    write_json(request_path, payload)
    status, headers, raw = post_json(f"{base_url}{endpoint}", payload)
    write_bytes(response_path, raw)
    body = decode_json_bytes(raw)
    summary = {
        "name": name,
        "endpoint": endpoint,
        "http_status": status,
        "content_type": headers.get("Content-Type") or headers.get("content-type"),
        "request_artifact": str(request_path),
        "response_artifact": str(response_path),
        "ok": 200 <= status < 300,
    }
    if endpoint.endswith("/chat/completions") and isinstance(body, dict):
        summary.update(summarize_chat_response(body))
    elif isinstance(body, dict):
        summary["id"] = body.get("id")
        summary["choices"] = body.get("choices")
        summary["usage"] = body.get("usage")
    return summary


def run_stream_case(base_url: str, artifacts: Path, name: str, endpoint: str, payload: dict[str, Any]) -> dict[str, Any]:
    request_path = artifacts / f"llm-proxy-gemini-{name}-request.json"
    response_path = artifacts / f"llm-proxy-gemini-{name}-sse.txt"
    payload = dict(payload)
    payload["stream"] = True
    write_json(request_path, payload)
    status, headers, raw = post_json(f"{base_url}{endpoint}", payload, timeout=180)
    write_bytes(response_path, raw)
    text = raw.decode("utf-8", errors="replace")
    return {
        "name": name,
        "endpoint": endpoint,
        "http_status": status,
        "content_type": headers.get("Content-Type") or headers.get("content-type"),
        "request_artifact": str(request_path),
        "response_artifact": str(response_path),
        "ok": 200 <= status < 300 and "data: [DONE]" in text,
        "contains_done": "data: [DONE]" in text,
        "data_line_count": sum(1 for line in text.splitlines() if line.startswith("data:")),
        "contains_tool_calls": "tool_calls" in text,
    }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--profile", default="gemini-3-flash-preview")
    parser.add_argument("--profiles", default=str(DEFAULT_PROFILES))
    parser.add_argument("--listen", default="127.0.0.1:0")
    parser.add_argument("--llm-proxy-repo", default=str(DEFAULT_LLM_PROXY_REPO))
    parser.add_argument("--artifacts", default=str(DEFAULT_ARTIFACTS))
    args = parser.parse_args()

    artifacts = Path(args.artifacts)
    artifacts.mkdir(parents=True, exist_ok=True)
    listen = allocate_listen_addr(args.listen)
    base_url = f"http://{listen}"
    repo = Path(args.llm_proxy_repo)
    profiles = Path(args.profiles)

    server_log = artifacts / "llm-proxy-gemini-server.log"
    with server_log.open("wb") as logf:
        proc = subprocess.Popen(
            ["go", "run", "./cmd/llm-proxy-server", "--profiles", str(profiles), "--listen", listen],
            cwd=repo,
            stdout=logf,
            stderr=subprocess.STDOUT,
        )
        try:
            wait_for_health(base_url, proc)
            cases: list[dict[str, Any]] = []

            status, _, models_raw = get_json(f"{base_url}/v1/models")
            models_path = artifacts / "llm-proxy-gemini-models-response.json"
            write_bytes(models_path, models_raw)
            models_body = decode_json_bytes(models_raw)
            model_ids = [m.get("id") for m in models_body.get("data", [])] if isinstance(models_body, dict) else []
            cases.append({"name": "models", "http_status": status, "ok": status == 200 and args.profile in model_ids, "response_artifact": str(models_path), "model_listed": args.profile in model_ids})

            cases.append(run_case(base_url, artifacts, "completions", "/v1/completions", {
                "model": args.profile,
                "prompt": "Reply with exactly: llm-proxy gemini completions ok",
                "max_tokens": 64,
            }))
            cases.append(run_stream_case(base_url, artifacts, "completions-stream", "/v1/completions", {
                "model": args.profile,
                "prompt": "Reply with exactly: llm-proxy gemini completions stream ok",
                "max_tokens": 64,
            }))
            cases.append(run_case(base_url, artifacts, "chat", "/v1/chat/completions", {
                "model": args.profile,
                "messages": [{"role": "user", "content": "Reply with exactly: llm-proxy gemini chat ok"}],
                "max_tokens": 64,
            }))
            cases.append(run_stream_case(base_url, artifacts, "chat-stream", "/v1/chat/completions", {
                "model": args.profile,
                "messages": [{"role": "user", "content": "Reply with exactly: llm-proxy gemini chat stream ok"}],
                "max_tokens": 64,
            }))

            tool_payload = {
                "model": args.profile,
                "messages": [{"role": "user", "content": "What is the weather in Zurich? Use lookup_weather."}],
                "tools": [{"type": "function", "function": {"name": "lookup_weather", "description": "Look up deterministic weather for a city.", "parameters": {"type": "object", "properties": {"city": {"type": "string", "description": "City name"}}, "required": ["city"]}}}],
            }
            tool_summary = run_case(base_url, artifacts, "tool-call", "/v1/chat/completions", tool_payload)
            cases.append(tool_summary)

            tool_calls = tool_summary.get("tool_calls") or []
            if tool_calls:
                tc = tool_calls[0]
                loop_payload = {
                    "model": args.profile,
                    "messages": [
                        {"role": "user", "content": "What is the weather in Zurich? Use lookup_weather."},
                        {"role": "assistant", "content": None, "tool_calls": [tc]},
                        {"role": "tool", "tool_call_id": tc["id"], "content": json.dumps({"city": "Zurich", "condition": "clear", "temperature": "21 C"})},
                        {"role": "user", "content": "Summarize the tool result in one sentence."},
                    ],
                    "tools": tool_payload["tools"],
                    "max_tokens": 128,
                }
                cases.append(run_case(base_url, artifacts, "tool-loop", "/v1/chat/completions", loop_payload))
            else:
                cases.append({"name": "tool-loop", "ok": False, "error": "tool-call case did not return tool_calls"})

            summary = {
                "profile": args.profile,
                "profiles_file": str(profiles),
                "base_url": base_url,
                "server_log": str(server_log),
                "cases": cases,
                "ok": all(c.get("ok") for c in cases),
            }
            summary_path = artifacts / "llm-proxy-gemini-smoke-summary.json"
            write_json(summary_path, summary)
            print(json.dumps(summary, indent=2))
            return 0 if summary["ok"] else 1
        finally:
            stop_process(proc)


if __name__ == "__main__":
    raise SystemExit(main())
