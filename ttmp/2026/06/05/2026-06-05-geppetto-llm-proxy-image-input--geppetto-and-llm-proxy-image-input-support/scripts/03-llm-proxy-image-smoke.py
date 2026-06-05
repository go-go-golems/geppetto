#!/usr/bin/env python3
import argparse
import json
import os
import signal
import socket
import subprocess
import sys
import time
from pathlib import Path
from urllib import request, error

TINY_PNG_DATA_URL = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII="


def repo_root() -> Path:
    return Path(__file__).resolve().parents[7]


def default_profiles() -> Path:
    home = Path.home()
    for p in [home / ".config/pinocchio/profiles.yaml", home / ".pinocchio/config/profiles.yaml"]:
        if p.exists():
            return p
    return home / ".config/pinocchio/profiles.yaml"


def allocate_listen(host: str = "127.0.0.1") -> str:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind((host, 0))
        return f"{host}:{sock.getsockname()[1]}"


def wait_for(base_url: str, proc: subprocess.Popen, timeout_s: float = 60) -> None:
    deadline = time.time() + timeout_s
    while time.time() < deadline:
        if proc.poll() is not None:
            raise RuntimeError(f"server exited with code {proc.returncode}")
        try:
            with request.urlopen(base_url + "/v1/models", timeout=2) as resp:
                if resp.status == 200:
                    return
        except Exception:
            time.sleep(0.25)
    raise TimeoutError(f"server did not become ready at {base_url}")


def post_json(base_url: str, path: str, payload: dict, out_path: Path) -> dict:
    body = json.dumps(payload).encode()
    req = request.Request(base_url + path, data=body, headers={"Content-Type": "application/json"}, method="POST")
    try:
        with request.urlopen(req, timeout=120) as resp:
            raw = resp.read().decode()
            out_path.write_text(raw + ("" if raw.endswith("\n") else "\n"))
            return {"http_status": resp.status, "response": json.loads(raw)}
    except error.HTTPError as e:
        raw = e.read().decode()
        out_path.write_text(raw + ("" if raw.endswith("\n") else "\n"))
        try:
            parsed = json.loads(raw)
        except Exception:
            parsed = raw
        return {"http_status": e.code, "response": parsed}


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--profile", default="gemini-3-flash-preview")
    parser.add_argument("--profiles", default=str(default_profiles()))
    parser.add_argument("--artifacts", default="ttmp/2026/06/05/2026-06-05-geppetto-llm-proxy-image-input--geppetto-and-llm-proxy-image-input-support/scripts/artifacts")
    args = parser.parse_args()

    root = repo_root()
    llm_proxy = root / "llm-proxy"
    artifacts = Path(args.artifacts)
    artifacts.mkdir(parents=True, exist_ok=True)
    listen = allocate_listen()
    base_url = f"http://{listen}"
    server_log = artifacts / "llm-proxy-image-server.log"

    with server_log.open("wb") as log:
        proc = subprocess.Popen(
            ["go", "run", "./cmd/llm-proxy-server", "--profiles", args.profiles, "--listen", listen],
            cwd=llm_proxy,
            stdout=log,
            stderr=subprocess.STDOUT,
            preexec_fn=os.setsid,
        )
        try:
            wait_for(base_url, proc)
            payload = {
                "model": args.profile,
                "messages": [{
                    "role": "user",
                    "content": [
                        {"type": "text", "text": "You are given an image. Reply exactly: image smoke ok"},
                        {"type": "image_url", "image_url": {"url": TINY_PNG_DATA_URL, "detail": "low"}},
                    ],
                }],
                "max_tokens": 32,
            }
            req_path = artifacts / "llm-proxy-image-chat-request.json"
            resp_path = artifacts / "llm-proxy-image-chat-response.json"
            req_path.write_text(json.dumps(payload, indent=2) + "\n")
            result = post_json(base_url, "/v1/chat/completions", payload, resp_path)
            response = result["response"]
            content = ""
            finish = ""
            if isinstance(response, dict) and response.get("choices"):
                choice = response["choices"][0]
                finish = choice.get("finish_reason", "")
                content = choice.get("message", {}).get("content", "")
            ok = result["http_status"] == 200 and "image smoke ok" in content.lower()
            summary = {
                "profile": args.profile,
                "base_url": base_url,
                "ok": ok,
                "http_status": result["http_status"],
                "finish_reason": finish,
                "content": content,
                "artifacts": {"request": str(req_path), "response": str(resp_path), "server_log": str(server_log)},
            }
            summary_path = artifacts / "llm-proxy-image-smoke-summary.json"
            summary_path.write_text(json.dumps(summary, indent=2) + "\n")
            print(json.dumps(summary, indent=2))
            return 0 if ok else 1
        finally:
            if proc.poll() is None:
                os.killpg(os.getpgid(proc.pid), signal.SIGTERM)
                try:
                    proc.wait(timeout=10)
                except subprocess.TimeoutExpired:
                    os.killpg(os.getpgid(proc.pid), signal.SIGKILL)


if __name__ == "__main__":
    sys.exit(main())
