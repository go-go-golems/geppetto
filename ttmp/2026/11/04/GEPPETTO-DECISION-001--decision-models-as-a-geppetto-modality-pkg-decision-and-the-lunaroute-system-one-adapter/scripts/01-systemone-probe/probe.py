# Probe LunaRoute System One. Reads the key from Pi's models.json; never prints it.
import json, os, sys, time, urllib.request, urllib.error
key = json.load(open(os.path.expanduser("~/.pi/agent/models.json")))["providers"]["lunaroute"]["apiKey"]
BASE = "https://gw.lunaroute.com"
def call(method, path, body=None, auth="lr"):
    h = {"content-type": "application/json"}
    if auth == "lr": h["LUNAROUTE-API-KEY"] = key
    else: h["Authorization"] = "Bearer " + key
    req = urllib.request.Request(BASE + path, method=method, headers=h, data=json.dumps(body).encode() if body is not None else None)
    t = time.time()
    try:
        with urllib.request.urlopen(req, timeout=60) as r:
            return r.status, json.loads(r.read()), (time.time()-t)*1000, dict(r.headers)
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode()[:600], (time.time()-t)*1000, dict(e.headers)
cmd = sys.argv[1]
if cmd == "models":
    s, b, ms, _ = call("GET", "/v1/models")
    print(s, round(ms))
    for m in b.get("data", []):
        caps = m.get("capabilities")
        if caps and (caps.get("systemone") or "systemone" in json.dumps(caps)):
            print(json.dumps(m, indent=1))
    print("all ids:", [m.get("id") for m in b.get("data", [])])
    print("example entry keys:", sorted(b["data"][0].keys()), json.dumps(b["data"][0].get("capabilities")))
elif cmd == "post":
    body = json.load(open(sys.argv[2]))
    s, b, ms, h = call("POST", "/v1/systemone", body, auth=sys.argv[3] if len(sys.argv) > 3 else "lr")
    print(s, round(ms), {k: v for k, v in h.items() if k.lower().startswith(("x-", "retry"))})
    print(json.dumps(b, indent=1) if isinstance(b, dict) else b)
