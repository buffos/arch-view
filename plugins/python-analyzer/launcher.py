"""NDJSON launcher for the Arch View external Python analyzer."""

from __future__ import annotations

import json
import sys

from archview_python import API_VERSION, MANIFEST, AnalysisFailure, analyze_request, detect


def write_frame(frame: dict) -> None:
    sys.stdout.write(json.dumps(frame, ensure_ascii=False, separators=(",", ":")) + "\n")
    sys.stdout.flush()


def fatal(request_id: str, failure: AnalysisFailure) -> None:
    write_frame({"type": "fatal", "request_id": request_id, "code": failure.code, "message": failure.message, "details": failure.details or None})


def main() -> int:
    write_frame({"type": "hello", "protocol": API_VERSION, "manifest": MANIFEST})
    line = sys.stdin.buffer.readline()
    if not line:
        return 1
    try:
        request = json.loads(line.decode("utf-8"))
    except (UnicodeDecodeError, json.JSONDecodeError) as exc:
        print(f"external Python analyzer request was not valid JSON: {exc}", file=sys.stderr)
        return 1
    request_id = str(request.get("request_id", ""))
    try:
        request_type = request.get("type")
        if request_type == "detect":
            candidate = detect(str(request.get("project_root", "")))
            write_frame({"type": "candidate", "request_id": request_id, "candidate": candidate})
            write_frame({"type": "done", "request_id": request_id, "status": "complete"})
            return 0
        if request_type == "analyze":
            options = request.get("options", {}).get("values", {})
            if not isinstance(options, dict):
                raise AnalysisFailure("invalid_payload", "analyze options must contain an object of values")
            fingerprint = str(request.get("options", {}).get("fingerprint", ""))
            result = analyze_request(str(request.get("project_root", "")), options, fingerprint)
            write_frame({"type": "result", "request_id": request_id, "result": result})
            write_frame({"type": "done", "request_id": request_id, "status": result["status"]})
            return 0
        if request_type == "cancel":
            write_frame({"type": "fatal", "request_id": request_id, "code": "cancelled", "message": "external Python analyzer cancelled"})
            return 0
        raise AnalysisFailure("invalid_payload", f"unsupported request type {request_type!r}")
    except AnalysisFailure as failure:
        fatal(request_id, failure)
        return 0
    except Exception as exc:  # noqa: BLE001 - process boundary converts failures to fatal frames
        print(f"external Python analyzer failed: {exc}", file=sys.stderr)
        fatal(request_id, AnalysisFailure("analyzer_failed", "external Python analyzer failed"))
        return 0


if __name__ == "__main__":
    raise SystemExit(main())
