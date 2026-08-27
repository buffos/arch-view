"""Static Python analysis for the Arch View external analyzer pilot.

This module intentionally uses only Python's standard library. It reads source
and project metadata as data, parses syntax with :mod:`ast`, and never imports,
executes, installs, or introspects the analyzed project.
"""

from __future__ import annotations

import ast
import hashlib
import io
import json
import os
import re
import tokenize
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any, Iterable


ANALYZER_ID = "org.archview.python.external"
ANALYZER_VERSION = "1.0.0"
API_VERSION = "arch-view.analyzer/v1"


MANIFEST: dict[str, Any] = {
    "id": ANALYZER_ID,
    "version": ANALYZER_VERSION,
    "language": "python",
    "api_version": API_VERSION,
    "detection_markers": [
        {"kind": "file", "value": "pyproject.toml", "weight": 1},
        {"kind": "file", "value": "setup.cfg", "weight": 0.9},
        {"kind": "file", "value": "setup.py", "weight": 0.8},
    ],
    "capabilities": ["detect", "static_dependencies", "dynamic_diagnostics"],
    "options": [
        {
            "name": "source_roots",
            "type": "string[]",
            "default": [],
            "description": "Explicit repository-relative Python source roots.",
            "sensitive": False,
        },
        {
            "name": "python_version",
            "type": "string",
            "default": None,
            "description": "Optional Python major/minor version used for static interpretation.",
            "sensitive": False,
        },
        {
            "name": "include_stubs",
            "type": "boolean",
            "default": False,
            "description": "Include .pyi stub files as module evidence.",
            "sensitive": False,
        },
        {
            "name": "include_tests",
            "type": "boolean",
            "default": False,
            "description": "Include Python test files and test directories.",
            "sensitive": False,
        },
        {
            "name": "exclude",
            "type": "string[]",
            "default": [],
            "description": "Additional repository-relative exclusion globs.",
            "sensitive": False,
        },
    ],
}


DEFAULT_EXCLUDED_DIRECTORIES = {
    ".git",
    ".hg",
    ".svn",
    ".cache",
    "__pycache__",
    ".mypy_cache",
    ".pytest_cache",
    ".tox",
    ".nox",
    "build",
    "cache",
    "dist",
    "generated",
    "out",
    "target",
    "tmp",
    "vendor",
    "external",
    "node_modules",
    ".venv",
    "venv",
    "env",
}


STDLIB_MODULES = set(
    """
    __future__ __main__ _thread abc aifc annotationlib antigravity argparse
    array asynchat asyncore ast asyncio atexit audioop base64 bdb binascii bisect builtins
    bz2 calendar cgi cgitb chunk cmath cmd code codecs codeop collections colorsys
    compileall compression concurrent configparser contextlib contextvars copy
    copyreg cProfile crypt csv ctypes curses dataclasses datetime dbm decimal
    difflib dis distutils doctest email encodings enum ensurepip errno faulthandler
    fcntl filecmp fileinput fnmatch fractions ftplib functools gc getopt getpass
    gettext glob graphlib grp gzip hashlib heapq hmac html http idlelib imaplib
    imghdr imp importlib inspect io ipaddress itertools json keyword lib2to3 linecache
    locale logging lzma mailbox mailcap marshal math mimetypes mmap modulefinder
    msilib msvcrt multiprocessing netrc nis nntplib numbers opcode operator
    optparse os ossaudiodev pathlib pdb pickle pickletools pipes pkgutil platform
    plistlib poplib posix pprint profile pstats pty pwd py_compile pyclbr pydoc
    pyexpat queue quopri random re readline reprlib resource rlcompleter runpy
    sched secrets select selectors shelve shlex shutil signal site smtpd smtplib
    sndhdr socket spwd socketserver sqlite3 ssl stat statistics string stringprep
    struct subprocess sunau symtable sys sysconfig syslog tabnanny tarfile telnetlib
    tempfile termios textwrap this threading time timeit tkinter token tokenize
    tomllib trace traceback tracemalloc tty turtle types typing unicodedata unittest
    urllib uu uuid venv warnings wave weakref webbrowser winreg winsound wsgiref
    xdrlib xml xmlrpc zipapp zipfile zipimport zlib zoneinfo
    """.split()
)
STDLIB_RULES: dict[str, tuple[int, int]] = {
    "contextvars": (7, 0),
    "annotationlib": (14, 0),
    "compression": (14, 0),
    "dataclasses": (7, 0),
    "graphlib": (9, 0),
    "ipaddress": (3, 0),
    "pathlib": (4, 0),
    "secrets": (6, 0),
    "statistics": (4, 0),
    "tomllib": (11, 0),
    "tracemalloc": (4, 0),
    "venv": (3, 0),
    "zoneinfo": (9, 0),
    "aifc": (0, 12),
    "asynchat": (0, 11),
    "asyncore": (0, 11),
    "audioop": (0, 12),
    "cgi": (0, 12),
    "cgitb": (0, 12),
    "chunk": (0, 12),
    "crypt": (0, 12),
    "distutils": (0, 11),
    "imghdr": (0, 12),
    "imp": (0, 11),
    "lib2to3": (0, 12),
    "mailcap": (0, 12),
    "msilib": (0, 12),
    "nis": (0, 12),
    "nntplib": (0, 12),
    "ossaudiodev": (0, 12),
    "pipes": (0, 12),
    "smtpd": (0, 11),
    "sndhdr": (0, 12),
    "spwd": (0, 12),
    "sunau": (0, 12),
    "telnetlib": (0, 12),
    "uu": (0, 12),
    "xdrlib": (0, 12),
}


VERSION_SELECTOR = re.compile(r"^3\.(?:[0-9]+|x)$")
VERSION_VALUE = re.compile(r"^3\.([0-9]+)$")
VERSION_IN_TEXT = re.compile(r"(?:^|[^0-9])([23]\.[0-9]+)(?:[^0-9]|$)")
VERSION_CONDITION = re.compile(r"^sys\.version_info\s*(>=|>|<=|<|==|!=)\s*\(\s*3\s*,\s*([0-9]+)\s*\)")


@dataclass
class SourceRoot:
    absolute: str
    relative: str


@dataclass
class Project:
    root: str
    boundary: str
    configuration_files: list[str]
    source_roots: list[SourceRoot]
    python_version: str
    config_diagnostics: list[dict[str, Any]]


@dataclass
class Token:
    kind: str
    text: str
    line: int
    column: int
    end_line: int
    end_column: int


@dataclass
class ImportObservation:
    from_module_id: str
    source: dict[str, Any]
    spelling: str
    module: str = ""
    imported_name: str = ""
    alias: str = ""
    relative_level: int = 0
    kind: str = "from"
    conditional: bool = False
    condition: str = ""
    dynamic_function: str = ""
    dynamic_target: str = ""


@dataclass
class FileObservation:
    path: str
    qualified: str
    kind: str
    source_id: str
    is_stub: bool
    is_test: bool
    source_root: str


@dataclass
class PackageObservation:
    qualified: str
    source_ids: set[str] = field(default_factory=set)
    paths: set[str] = field(default_factory=set)
    source_roots: set[str] = field(default_factory=set)
    init_source_roots: set[str] = field(default_factory=set)
    has_init: bool = False
    tags: set[str] = field(default_factory=set)


@dataclass
class ModuleObservation:
    qualified: str
    source_ids: set[str] = field(default_factory=set)
    paths: set[str] = field(default_factory=set)
    tags: set[str] = field(default_factory=set)


@dataclass
class Discovery:
    modules: list[dict[str, Any]]
    imports: list[ImportObservation]
    source_references: list[dict[str, Any]]
    diagnostics: list[dict[str, Any]]
    ambiguous_names: dict[str, bool]


def stable_id(kind: str, *values: str) -> str:
    payload = (kind + "\0" + "\0".join(values)).encode("utf-8")
    return "py:" + kind + ":" + hashlib.sha256(payload).hexdigest()[:16]


def module_id(kind: str, qualified: str) -> str:
    return f"py:{kind}:{qualified}"


def json_key(value: Any) -> str:
    return json.dumps(value, ensure_ascii=False, sort_keys=True, separators=(",", ":"))


def quoted(value: str) -> str:
    return json.dumps(value, ensure_ascii=False)


def diagnostic_sort_key(value: dict[str, Any]) -> str:
    return json_key(value)


def normalize_path(path: str) -> str:
    return path.replace("\\", "/")


def path_within(root: str, candidate: str) -> bool:
    try:
        return os.path.commonpath([os.path.abspath(root), os.path.abspath(candidate)]) == os.path.abspath(root)
    except ValueError:
        return False


def resolved_path_within(root: str, candidate: str) -> bool:
    try:
        return path_within(os.path.realpath(root), os.path.realpath(candidate))
    except OSError:
        return False


def relative_project_path(root: str, path: str) -> str:
    try:
        return normalize_path(os.path.relpath(path, root))
    except ValueError:
        return normalize_path(os.path.normpath(path))


def exists_as_file(path: str) -> bool:
    try:
        return os.path.isfile(path)
    except OSError:
        return False


def safe_project_file(root: str, relative: str) -> bool:
    path = os.path.join(root, relative.replace("/", os.sep))
    return exists_as_file(path) and resolved_path_within(root, path)


def preferred_boundary(root: str) -> str:
    for marker in ("pyproject.toml", "setup.cfg", "setup.py"):
        if safe_project_file(root, marker):
            return marker
    return ""


def option_string(options: dict[str, Any], name: str) -> str:
    value = options.get(name)
    return value.strip() if isinstance(value, str) else ""


def option_strings(options: dict[str, Any], name: str) -> list[str]:
    value = options.get(name, [])
    if not isinstance(value, list):
        return []
    return [item for item in value if isinstance(item, str)]


def option_bool(options: dict[str, Any], name: str) -> bool:
    return options.get(name) is True


def configuration_diagnostic(code: str, path: str, line: int, message: str) -> dict[str, Any]:
    if line > 0:
        message = f"{message} (line {line})"
    return {"code": code, "severity": "warning", "message": message, "path": path, "recoverable": True}


def first_python_version(value: str) -> str:
    match = VERSION_IN_TEXT.search(value)
    return match.group(1) if match else ""


def read_configuration(root: str, boundary: str) -> tuple[list[str], str, list[dict[str, Any]]]:
    path = os.path.join(root, boundary)
    if not safe_project_file(root, boundary):
        return [], "", [configuration_diagnostic("python_configuration_outside_root", boundary, 0, "Python project configuration resolves outside the selected project root and was ignored.")]
    try:
        content = Path(path).read_text(encoding="utf-8")
    except OSError as exc:
        return [], "", [{"code": "python_configuration_unreadable", "severity": "warning", "message": f"Python project configuration {boundary!r} could not be read: {exc}", "path": boundary, "recoverable": True}]
    if boundary == "pyproject.toml":
        return parse_pyproject(content, boundary)
    if boundary == "setup.cfg":
        return parse_setup_cfg(content, boundary)
    return parse_setup_py(content, boundary)


def parse_pyproject(content: str, boundary: str) -> tuple[list[str], str, list[dict[str, Any]]]:
    diagnostics: list[dict[str, Any]] = []
    try:
        try:
            import tomllib

            data = tomllib.loads(content)
        except ModuleNotFoundError:
            data = parse_minimal_toml(content)
    except Exception as exc:  # noqa: BLE001 - malformed project data is recoverable
        line = getattr(exc, "lineno", 0) or 0
        diagnostics.append(configuration_diagnostic("python_configuration_invalid", boundary, line, f"Python project configuration could not be parsed safely: {exc}"))
        data = {}

    roots: list[str] = []
    def append_values(value: Any) -> None:
        if isinstance(value, str):
            roots.append(value)
        elif isinstance(value, list):
            roots.extend(item for item in value if isinstance(item, str))

    for key in ("source_roots", "source-roots"):
        append_values(data.get(key))
    setuptools = data.get("tool", {}).get("setuptools", {}) if isinstance(data.get("tool"), dict) else {}
    package_find = setuptools.get("packages", {}).get("find", {}) if isinstance(setuptools.get("packages"), dict) else {}
    append_values(package_find.get("where"))
    package_dir = setuptools.get("package-dir")
    if package_dir is None:
        package_dir = setuptools.get("package_dir")
    if isinstance(package_dir, dict):
        value = package_dir.get("")
        if isinstance(value, str):
            roots.append(value)
    elif isinstance(package_dir, str):
        roots.append(package_dir)
    poetry_packages = data.get("tool", {}).get("poetry", {}).get("packages", []) if isinstance(data.get("tool"), dict) else []
    if isinstance(poetry_packages, list):
        for package in poetry_packages:
            if isinstance(package, dict) and isinstance(package.get("from"), str):
                roots.append(package["from"])
    project = data.get("project", {})
    python_version = first_python_version(project.get("requires-python", "")) if isinstance(project, dict) and isinstance(project.get("requires-python"), str) else ""
    return roots, python_version, diagnostics


def parse_minimal_toml(content: str) -> dict[str, Any]:
    # Python 3.10 fallback. It covers the safe string/list/table forms used by
    # the project contract and deliberately ignores executable TOML extensions.
    data: dict[str, Any] = {}
    section = data
    for raw in content.replace("\r\n", "\n").splitlines():
        line = raw.split("#", 1)[0].strip()
        if not line:
            continue
        if line.startswith("[") and line.endswith("]"):
            section = data
            for part in line[1:-1].strip().split("."):
                section = section.setdefault(part, {})
            continue
        if "=" not in line:
            continue
        key, raw_value = (part.strip() for part in line.split("=", 1))
        section[key.strip('"')] = parse_toml_value(raw_value)
    return data


def parse_toml_value(value: str) -> Any:
    value = value.strip()
    if value.startswith("[") and value.endswith("]"):
        return [parse_toml_value(item.strip()) for item in value[1:-1].split(",") if item.strip()]
    if len(value) >= 2 and value[0] in "'\"" and value[-1] == value[0]:
        return value[1:-1]
    return value


def parse_setup_cfg(content: str, boundary: str) -> tuple[list[str], str, list[dict[str, Any]]]:
    roots: list[str] = []
    python_version = ""
    diagnostics: list[dict[str, Any]] = []
    section = ""
    last_key = ""
    last_section = ""
    for number, raw in enumerate(content.replace("\r\n", "\n").splitlines(), 1):
        trimmed = raw.strip()
        if not trimmed or trimmed.startswith(";") or trimmed.startswith("#"):
            continue
        if trimmed.startswith("[") and trimmed.endswith("]"):
            section = trimmed[1:-1].strip().lower()
            last_key = ""
            last_section = ""
            continue
        if raw[:1] in (" ", "\t") and last_key:
            if last_section == "options" and last_key.replace("-", "_").lower() == "package_dir":
                roots.extend(parse_setup_package_dir(trimmed))
            continue
        separator = next((index for index in (trimmed.find("="), trimmed.find(":")) if index >= 0), -1)
        if separator < 0:
            diagnostics.append(configuration_diagnostic("python_configuration_invalid", boundary, number, "Python setup.cfg contains a setting without a key/value separator."))
            continue
        key = trimmed[:separator].strip()
        value = trimmed[separator + 1 :].strip()
        last_key, last_section = key, section
        normalized = key.lower().replace("-", "_")
        if section == "options.packages.find" and normalized == "where":
            roots.extend(split_config_roots(value))
        elif section == "options" and normalized == "package_dir":
            roots.extend(parse_setup_package_dir(value))
        elif section == "options" and normalized == "python_requires":
            python_version = first_python_version(value)
    return roots, python_version, diagnostics


def parse_setup_package_dir(value: str) -> list[str]:
    value = value.strip()
    if value.startswith("="):
        value = value[1:].strip()
    if not value:
        return []
    return [value.strip("\"'")]


def split_config_roots(value: str) -> list[str]:
    value = value.replace(",", " ")
    return [field.strip("\"'") for field in value.split() if field.strip("\"'")]


def parse_setup_py(content: str, boundary: str) -> tuple[list[str], str, list[dict[str, Any]]]:
    diagnostics: list[dict[str, Any]] = []
    try:
        ast.parse(content)
    except SyntaxError as exc:
        diagnostics.append(configuration_diagnostic("python_configuration_invalid", boundary, exc.lineno or 0, "Python setup.py could not be parsed as static configuration data."))
    roots: list[str] = []
    for match in re.finditer(r"(?s)package_dir\s*=\s*\{.*?['\"]{2}\s*:\s*['\"]([^'\"]+)['\"]", content):
        roots.append(match.group(1).strip())
    for match in re.finditer(r"(?s)find_(?:namespace_)?packages\s*\((.*?)\)", content):
        where = re.search(r"['\"]?where['\"]?\s*=\s*['\"]([^'\"]+)['\"]", match.group(1))
        if where:
            roots.append(where.group(1).strip())
    return roots, first_python_version(content), diagnostics


def normalize_source_root(root: str, value: str) -> tuple[SourceRoot | None, str | None]:
    trimmed = value.strip()
    if not trimmed:
        return None, "source root is empty"
    absolute = os.path.normpath(trimmed if os.path.isabs(trimmed) else os.path.join(root, trimmed))
    if not path_within(root, absolute):
        return None, "source root escapes the selected project root"
    if not os.path.isdir(absolute):
        return None, "source root is not a directory"
    if not resolved_path_within(root, absolute):
        return None, "source root resolves outside the selected project root"
    relative = normalize_path(os.path.relpath(absolute, root))
    return SourceRoot(absolute, "." if relative == "." else relative), None


def resolve_project(root: str, options: dict[str, Any]) -> Project:
    absolute_root = os.path.normpath(os.path.abspath(root))
    if not os.path.isdir(absolute_root):
        raise AnalysisFailure("unreadable_project", "Python project root does not exist or is not a directory")
    boundary = preferred_boundary(absolute_root)
    if not boundary:
        raise AnalysisFailure("unsupported_project", "Python project requires pyproject.toml, setup.cfg, or setup.py at the selected root")
    configured_roots, configured_version, diagnostics = read_configuration(absolute_root, boundary)
    explicit_roots = option_strings(options, "source_roots")
    root_values = explicit_roots or configured_roots
    root_source = "explicit analyzer options" if explicit_roots else "project configuration"
    if not root_values:
        if os.path.isdir(os.path.join(absolute_root, "src")):
            root_values, root_source = ["src"], "src layout default"
        else:
            root_values, root_source = ["."], "project root default"
    roots: list[SourceRoot] = []
    for value in root_values:
        source_root, error = normalize_source_root(absolute_root, value)
        if error:
            diagnostics.append({"code": "python_source_root_invalid", "severity": "warning", "message": f"Python source root {value!r} was ignored: {error}", "subject": value, "recoverable": True})
        elif source_root and not any(item.absolute == source_root.absolute for item in roots):
            roots.append(source_root)
    if not roots:
        fallback, error = normalize_source_root(absolute_root, ".")
        if fallback:
            roots.append(fallback)
            diagnostics.append({"code": "python_source_root_fallback", "severity": "warning", "message": f"No usable source roots were found from {root_source}; the project root was used as a safe fallback.", "recoverable": True})
    roots.sort(key=lambda item: item.relative)
    python_version = option_string(options, "python_version") or configured_version
    if python_version and not VERSION_SELECTOR.match(python_version.strip()):
        diagnostics.append({"code": "python_unsupported_version", "severity": "warning", "message": f"Python version {python_version!r} is not a supported static version selector; discovery continued without executing Python.", "subject": python_version, "recoverable": True})
        python_version = ""
    return Project(absolute_root, boundary, [boundary], roots, python_version, diagnostics)


def is_test_path(relative: str, name: str) -> bool:
    lower = name.lower()
    stem = lower
    if stem.endswith(".pyi"):
        stem = stem[:-4]
    if stem.endswith(".py"):
        stem = stem[:-3]
    if lower in {"conftest.py", "conftest.pyi", "test.py"} or stem.startswith("test_") or stem.endswith("_test"):
        return True
    return any(segment in {"test", "tests", "testing"} for segment in normalize_path(relative).lower().split("/"))


def excluded_file_path(relative: str, patterns: Iterable[str]) -> bool:
    clean = normalize_path(os.path.normpath(relative))
    for pattern in patterns:
        pattern = normalize_path(os.path.normpath(pattern)).lstrip("./")
        if not pattern or pattern == ".":
            continue
        if clean == pattern.removesuffix("/**"):
            return True
        if glob_match(pattern, clean):
            return True
        if pattern.endswith("/**") and clean.startswith(pattern[:-3].rstrip("/") + "/"):
            return True
        if pattern.startswith("**/") and glob_match(pattern[3:], clean.split("/")[-1]):
            return True
    return False


def glob_match(pattern: str, value: str) -> bool:
    regex = re.escape(pattern).replace(r"\*\*", ".*").replace(r"\*", "[^/]*").replace(r"\?", "[^/]")
    return re.fullmatch(regex, value) is not None


def excluded_directory(relative: str, patterns: Iterable[str]) -> bool:
    clean = normalize_path(os.path.normpath(relative)).strip("/")
    if not clean or clean == ".":
        return False
    if any(segment.lower() in DEFAULT_EXCLUDED_DIRECTORIES for segment in clean.split("/")):
        return True
    return excluded_file_path(clean, patterns)


def source_qualified_name(source_root: str, file_path: str, name: str) -> tuple[str, str]:
    relative = os.path.relpath(os.path.dirname(file_path), source_root)
    parts = [] if relative == "." else [part for part in normalize_path(relative).split("/") if part not in {"", "."}]
    base = name[: -len(Path(name).suffix)] if Path(name).suffix else name
    if base == "__init__":
        return ("__root__", "package") if not parts else (".".join(parts), "package")
    parts.append(base)
    return ".".join(parts), "module"


def source_reference(path: str, start: Token, end: Token, symbol: str, kind: str) -> dict[str, Any]:
    return {
        "id": stable_id("import", path, str(start.line), str(start.column), str(end.end_line), str(end.end_column), symbol, kind),
        "path": path,
        "start": {"line": start.line, "column": start.column},
        "end": {"line": end.end_line, "column": end.end_column},
        "symbol": symbol,
        "kind": kind,
    }


def lex_python(content: str) -> list[Token]:
    tokens: list[Token] = []
    stream = tokenize.generate_tokens(io.StringIO(content).readline)
    while True:
        try:
            item = next(stream)
        except StopIteration:
            break
        except (tokenize.TokenError, IndentationError):
            break
        kind, text, start, end, _ = item
        line, column = start[0], start[1] + 1
        end_line, end_column = end[0], end[1] + 1
        if kind == tokenize.NAME:
            tokens.append(Token("name", text, line, column, end_line, end_column))
        elif kind == tokenize.STRING:
            tokens.append(Token("string", text, line, column, end_line, end_column))
        elif kind == tokenize.OP:
            tokens.append(Token("punctuation", text, line, column, end_line, end_column))
        elif kind in (tokenize.NEWLINE, tokenize.NL):
            tokens.append(Token("newline", "\n", line, column, end_line, end_column))
    return tokens


def conditional_contexts(content: str) -> dict[int, list[str]]:
    result: dict[int, list[str]] = {}
    stack: list[tuple[int, str]] = []
    lines = content.replace("\r\n", "\n").split("\n")
    pending_header = ""
    pending_indent = 0
    pending_contexts: list[str] = []
    for number, raw in enumerate(lines, 1):
        trimmed = raw.strip()
        if not trimmed or trimmed.startswith("#"):
            continue
        if pending_header:
            pending_header += " " + normalize_conditional_line(trimmed)
            result[number] = list(pending_contexts)
            header, block = conditional_header(pending_header)
            if header:
                if block:
                    stack.append((pending_indent, header))
                else:
                    result[number].append(header)
                pending_header = ""
                pending_contexts = []
            continue
        indent = python_indent(raw)
        while stack and indent <= stack[-1][0]:
            stack.pop()
        values = [frame[1] for frame in stack]
        result[number] = values
        header, block = conditional_header(trimmed)
        if header:
            if block:
                stack.append((indent, header))
            else:
                result[number].append(header)
        elif conditional_header_can_continue(trimmed):
            pending_header = normalize_conditional_line(trimmed)
            pending_indent = indent
            pending_contexts = list(values)
    return result


def conditional_header_can_continue(line: str) -> bool:
    lower = line.strip().lower()
    return lower.startswith(("if ", "elif ", "while ", "for ", "except ", "case "))


def normalize_conditional_line(line: str) -> str:
    return code_before_comment(line).strip().removesuffix("\\").strip()


def code_before_comment(line: str) -> str:
    quote = ""
    index = 0
    while index < len(line):
        char = line[index]
        if quote:
            if char == "\\":
                index += 2
                continue
            if char == quote:
                quote = ""
        elif char in "'\"":
            quote = char
        elif char == "#":
            return line[:index]
        index += 1
    return line


def python_indent(value: str) -> int:
    indent = 0
    for char in value:
        if char == " ":
            indent += 1
        elif char == "\t":
            indent += 8 - indent % 8
        else:
            break
    return indent


def conditional_header(line: str) -> tuple[str, bool]:
    colon = suite_colon(line)
    if colon < 0:
        return "", False
    prefix = line[:colon].strip()
    lower = prefix.lower()
    conditional = lower in {"try", "else", "finally"} or lower.startswith(("if ", "elif ", "while ", "for ", "except", "case "))
    if not conditional:
        return "", False
    rest = line[colon + 1 :].strip()
    if "#" in rest:
        rest = rest.split("#", 1)[0].strip()
    return prefix + ":", not rest


def suite_colon(line: str) -> int:
    quote = ""
    depth = 0
    index = 0
    while index < len(line):
        char = line[index]
        if quote:
            if char == "\\":
                index += 2
                continue
            if char == quote:
                quote = ""
        elif char in "'\"":
            quote = char
        elif char == "#":
            break
        elif char in "([{":
            depth += 1
        elif char in ")]}":
            depth = max(depth - 1, 0)
        elif char == ":" and depth == 0:
            return index
        index += 1
    return -1


def next_token(tokens: list[Token], index: int) -> int:
    while index < len(tokens) and tokens[index].kind == "newline":
        index += 1
    return index


def token_text(tokens: Iterable[Token]) -> str:
    return "".join(token.text for token in tokens if token.kind != "newline")


def make_static_observation(path: str, from_id: str, start: Token, end: Token, spelling: str, module: str, imported: str, kind: str, level: int, conditions: dict[int, list[str]], alias: str = "") -> ImportObservation:
    values = list(conditions.get(start.line, []))
    condition = " -> ".join(values)
    return ImportObservation(from_id, source_reference(path, start, end, spelling, "import"), spelling, module, imported, alias, level, kind, bool(condition), condition)


def parse_import_statement(tokens: list[Token], start: int, path: str, from_id: str, conditions: dict[int, list[str]]) -> tuple[list[ImportObservation], int]:
    values: list[ImportObservation] = []
    index = start + 1
    while index < len(tokens):
        if tokens[index].kind == "newline" or tokens[index].text == ";":
            break
        if tokens[index].text == "(":
            index += 1
            continue
        if tokens[index].kind != "name":
            break
        first = tokens[index]
        parts = [first.text]
        last = first
        index += 1
        while index + 1 < len(tokens) and tokens[index].text == "." and tokens[index + 1].kind == "name":
            parts.append(tokens[index + 1].text)
            last = tokens[index + 1]
            index += 2
        spelling = ".".join(parts)
        alias = ""
        if index < len(tokens) and tokens[index].kind == "name" and tokens[index].text == "as":
            index += 1
            if index < len(tokens) and tokens[index].kind == "name":
                alias = tokens[index].text
                index += 1
        values.append(make_static_observation(path, from_id, first, last, spelling, spelling, "", "import", 0, conditions, alias))
        if index >= len(tokens) or tokens[index].text != ",":
            break
        index += 1
    return values, index


def parse_from_statement(tokens: list[Token], start: int, path: str, from_id: str, package_init: bool, conditions: dict[int, list[str]]) -> tuple[list[ImportObservation], int, bool]:
    index = start + 1
    level = 0
    module_start: Token | None = None
    module_parts: list[str] = []
    while index < len(tokens) and tokens[index].text == ".":
        module_start = module_start or tokens[index]
        level += 1
        index += 1
    while index < len(tokens) and tokens[index].kind == "name" and tokens[index].text != "import":
        module_start = module_start or tokens[index]
        module_parts.append(tokens[index].text)
        index += 1
        if index >= len(tokens) or tokens[index].text != "." or (index + 1 < len(tokens) and tokens[index + 1].kind != "name"):
            break
        index += 1
    if index >= len(tokens) or tokens[index].kind != "name" or tokens[index].text != "import":
        return [], index, False
    index += 1
    values: list[ImportObservation] = []
    depth = 0
    while index < len(tokens):
        if tokens[index].kind == "newline":
            if depth > 0:
                index += 1
                continue
            break
        if tokens[index].text == "(":
            depth += 1
            index += 1
            continue
        if tokens[index].text == ")":
            if depth == 0:
                break
            depth -= 1
            index += 1
            continue
        if tokens[index].text == ",":
            index += 1
            continue
        if tokens[index].kind != "name" and tokens[index].text != "*":
            break
        imported_token = tokens[index]
        imported = imported_token.text
        index += 1
        alias = ""
        if index < len(tokens) and tokens[index].kind == "name" and tokens[index].text == "as":
            index += 1
            if index < len(tokens) and tokens[index].kind == "name":
                alias = tokens[index].text
                index += 1
        module = ".".join(module_parts)
        spelling = "." * level + (module + "." if module else "") + imported
        kind = "reexport" if package_init else "from"
        values.append(make_static_observation(path, from_id, module_start or tokens[start], imported_token, spelling, module, imported, kind, level, conditions, alias))
        if index >= len(tokens) or tokens[index].text != ",":
            break
        index += 1
    return values, index, True


def static_string(value: str) -> tuple[str, bool]:
    try:
        parsed = ast.literal_eval(value)
    except (ValueError, SyntaxError):
        return "", False
    if isinstance(parsed, str):
        return parsed, True
    return "", False


KNOWN_DYNAMIC_CALLABLES = {
    "__import__",
    "builtins.__import__",
    "importlib.import_module",
    "importlib.util.find_spec",
    "importlib.util.module_from_spec",
    "importlib.metadata.entry_points",
    "pkg_resources.iter_entry_points",
    "pkg_resources.load_entry_point",
    "pkg_resources.load_setuptools_entrypoints",
    "stevedore.extension.ExtensionManager",
}


def matching_call_end(tokens: list[Token], opening: int) -> int:
    depth = 0
    for index in range(opening, len(tokens)):
        if tokens[index].text == "(":
            depth += 1
        elif tokens[index].text == ")":
            depth -= 1
            if depth == 0:
                return index
    return opening


def static_call_target(tokens: list[Token], argument: int, call_end: int) -> str:
    if argument >= len(tokens) or argument >= call_end or tokens[argument].kind != "string":
        return ""
    values: list[str] = []
    cursor = argument
    while cursor < call_end and tokens[cursor].kind == "string":
        parsed, ok = static_string(tokens[cursor].text)
        if not ok:
            return ""
        values.append(parsed)
        cursor = next_token(tokens, cursor + 1)
    if cursor < call_end and tokens[cursor].text != ",":
        return ""
    return "".join(values)


def register_dynamic_aliases(aliases: set[str], observations: Iterable[ImportObservation]) -> None:
    for observation in observations:
        if observation.kind == "import":
            binding = observation.alias or (observation.module if "." not in observation.module else "")
            register_dynamic_module_binding(aliases, binding, observation.module)
            continue
        canonical = join_qualified(observation.module, observation.imported_name)
        name = observation.alias or observation.imported_name
        if canonical in KNOWN_DYNAMIC_CALLABLES:
            aliases.add(name)
        register_dynamic_module_binding(aliases, name, canonical)


def register_dynamic_module_binding(aliases: set[str], binding: str, module: str) -> None:
    if not binding:
        return
    names = {
        "importlib": ["import_module"],
        "importlib.util": ["find_spec", "module_from_spec"],
        "importlib.metadata": ["entry_points"],
        "pkg_resources": ["iter_entry_points", "load_entry_point", "load_setuptools_entrypoints"],
        "stevedore.extension": ["ExtensionManager"],
    }.get(module, [])
    aliases.update(f"{binding}.{name}" for name in names)


def parse_dynamic_call(tokens: list[Token], index: int, path: str, from_id: str, conditions: dict[int, list[str]], aliases: set[str]) -> tuple[ImportObservation | None, int]:
    if tokens[index].kind != "name":
        return None, index
    if index > 0 and tokens[index - 1].kind == "name" and tokens[index - 1].text in {"def", "class"}:
        return None, index
    opening = next_token(tokens, index + 1)
    if opening >= len(tokens) or tokens[opening].text != "(":
        return None, index
    start = index
    while start >= 2 and tokens[start - 1].text == "." and tokens[start - 2].kind == "name":
        start -= 2
    function = token_text(tokens[start : index + 1])
    if function not in KNOWN_DYNAMIC_CALLABLES and function not in aliases:
        return None, index
    end = matching_call_end(tokens, opening)
    argument = next_token(tokens, opening + 1)
    target = static_call_target(tokens, argument, end)
    name = target or "<dynamic>"
    end_token = tokens[end] if end < len(tokens) else tokens[opening]
    condition = " -> ".join(conditions.get(tokens[start].line, []))
    return ImportObservation(
        from_id,
        source_reference(path, tokens[start], end_token, name, "dynamic"),
        name,
        kind="dynamic",
        conditional=bool(condition),
        condition=condition,
        dynamic_function=function,
        dynamic_target=target,
    ), end


def extract_imports(path: str, content: str, from_id: str, package_init: bool) -> tuple[list[ImportObservation], list[dict[str, Any]]]:
    # ast.parse is deliberately part of the external boundary. The lexical
    # pass below preserves the baseline's conservative import evidence even
    # when a file has a recoverable syntax problem.
    tokens = lex_python(content)
    conditions = conditional_contexts(content)
    observations: list[ImportObservation] = []
    diagnostics: list[dict[str, Any]] = []
    aliases: set[str] = set()
    statement_start = True
    bracket_depth = 0
    index = 0
    while index < len(tokens):
        token = tokens[index]
        if token.kind == "newline":
            if bracket_depth == 0:
                statement_start = True
            index += 1
            continue
        dynamic, end = parse_dynamic_call(tokens, index, path, from_id, conditions, aliases)
        if dynamic is not None:
            observations.append(dynamic)
            index = end + 1
            statement_start = False
            continue
        if statement_start and token.kind == "name" and token.text == "import":
            parsed, next_index = parse_import_statement(tokens, index, path, from_id, conditions)
            observations.extend(parsed)
            register_dynamic_aliases(aliases, parsed)
            index = max(next_index, index + 1)
            statement_start = False
            continue
        if statement_start and token.kind == "name" and token.text == "from":
            parsed, next_index, valid = parse_from_statement(tokens, index, path, from_id, package_init, conditions)
            observations.extend(parsed)
            register_dynamic_aliases(aliases, parsed)
            if not valid:
                diagnostics.append({"code": "python_import_syntax", "severity": "warning", "message": "Python from-import could not be interpreted statically; unrelated observations were retained.", "path": path, "location": {"line": token.line, "column": token.column}, "recoverable": True})
            index = max(next_index, index + 1)
            statement_start = False
            continue
        if token.text in "([{":
            bracket_depth += 1
        elif token.text in ")]}":
            bracket_depth = max(bracket_depth - 1, 0)
        elif token.text in {";", ":"} and bracket_depth == 0:
            statement_start = True
        else:
            statement_start = False
        index += 1
    return observations, diagnostics


def syntax_diagnostic(path: str, content: str) -> dict[str, Any] | None:
    try:
        ast.parse(content, filename=path)
        return None
    except (SyntaxError, ValueError, TypeError) as exc:
        line = getattr(exc, "lineno", 1) or 1
        column = getattr(exc, "offset", 1) or 1
        return {"code": "python_syntax_error", "severity": "error", "message": "Python source has an unsupported or malformed syntax construct; its module evidence was retained.", "path": path, "location": {"line": line, "column": column}, "recoverable": True}


def add_file_tags(tags: set[str], file: FileObservation) -> None:
    if file.is_stub:
        tags.add("stub")
    if file.is_test:
        tags.add("test")


def stub_companion_paths(first: str, second: str) -> bool:
    first_ext, second_ext = Path(first).suffix.lower(), Path(second).suffix.lower()
    return {first_ext, second_ext} == {".py", ".pyi"} and Path(first).stem == Path(second).stem


def first_conflicting_path(paths: set[str], current: str) -> str:
    for value in sorted(paths):
        if value != current and not stub_companion_paths(value, current):
            return value
    return ""


def parent_packages(qualified: str) -> list[str]:
    parts = qualified.split(".")
    return [".".join(parts[: index + 1]) for index in range(len(parts) - 1) if parts[index] not in {"", "__root__"}]


def build_module_observations(project: Project, packages: dict[str, PackageObservation], modules: dict[str, ModuleObservation]) -> list[dict[str, Any]]:
    root_values = sorted(root.relative for root in project.source_roots)
    result: list[dict[str, Any]] = []
    for qualified in sorted(packages):
        value = packages[qualified]
        package_kind = "regular" if value.has_init else "namespace"
        if not value.has_init:
            value.tags.add("namespace")
        paths = sorted(value.paths)
        metadata: dict[str, Any] = {
            "qualified_name": qualified,
            "package_kind": package_kind,
            "paths": paths,
            "file_count": len(paths),
            "source_roots": root_values,
            "configuration_files": list(project.configuration_files),
        }
        if project.python_version:
            metadata["python_version"] = project.python_version
        result.append({
            "id": module_id("package", qualified),
            "language": "python",
            "kind": "package",
            "name": qualified.split(".")[-1],
            "display_name": qualified,
            "hierarchy": qualified.split(".") if qualified else [],
            "source_reference_ids": sorted(value.source_ids),
            "tags": sorted(value.tags),
            "metadata": metadata,
        })
    for qualified in sorted(modules):
        value = modules[qualified]
        paths = sorted(value.paths)
        metadata = {
            "qualified_name": qualified,
            "module_kind": "module",
            "relative_paths": paths,
            "relative_path": paths[0],
            "file_count": len(paths),
            "source_roots": root_values,
            "configuration_files": list(project.configuration_files),
            "stub": "stub" in value.tags,
        }
        if project.python_version:
            metadata["python_version"] = project.python_version
        result.append({
            "id": module_id("module", qualified),
            "language": "python",
            "kind": "module",
            "name": qualified.split(".")[-1],
            "display_name": qualified,
            "hierarchy": qualified.split(".") if qualified else [],
            "source_reference_ids": sorted(value.source_ids),
            "tags": sorted(value.tags),
            "metadata": metadata,
        })
    result.sort(key=lambda item: item["id"])
    return result


def discover(project: Project, options: dict[str, Any]) -> Discovery:
    include_stubs = option_bool(options, "include_stubs")
    include_tests = option_bool(options, "include_tests")
    exclude_patterns = option_strings(options, "exclude")
    packages: dict[str, PackageObservation] = {}
    modules: dict[str, ModuleObservation] = {}
    sources: dict[str, dict[str, Any]] = {}
    files: dict[str, FileObservation] = {}
    imports: list[ImportObservation] = []
    diagnostics = list(project.config_diagnostics)
    ambiguous: dict[str, bool] = {}

    for source_root in project.source_roots:
        if excluded_directory(source_root.relative, exclude_patterns):
            continue
        for current, directories, filenames in os.walk(source_root.absolute, followlinks=False):
            directories[:] = sorted(directory for directory in directories if not excluded_directory(relative_project_path(project.root, os.path.join(current, directory)), exclude_patterns))
            for name in sorted(filenames):
                extension = Path(name).suffix.lower()
                if extension != ".py" and not (include_stubs and extension == ".pyi"):
                    continue
                relative = relative_project_path(project.root, os.path.join(current, name))
                if excluded_file_path(relative, exclude_patterns) or (not include_tests and is_test_path(relative, name)):
                    continue
                absolute_path = os.path.join(current, name)
                if not path_within(project.root, absolute_path) or not resolved_path_within(project.root, absolute_path):
                    diagnostics.append({"code": "python_path_outside_root", "severity": "warning", "message": "Python source file resolves outside the selected project root and was ignored.", "path": relative, "recoverable": True})
                    continue
                try:
                    raw = Path(absolute_path).read_bytes()
                    content = raw.decode("utf-8")
                    valid_utf8 = True
                except UnicodeDecodeError:
                    raw = Path(absolute_path).read_bytes()
                    content = raw.decode("utf-8", errors="replace")
                    valid_utf8 = False
                except OSError as exc:
                    diagnostics.append({"code": "python_unreadable_file", "severity": "error", "message": f"Python source path could not be read: {exc}", "path": relative, "recoverable": True})
                    continue
                issue = None if valid_utf8 else {"line": 1, "column": 1}
                if issue is None:
                    issue = syntax_diagnostic(relative, content)
                if issue:
                    diagnostics.append({"code": "python_syntax_error", "severity": "error", "message": "Python source has an unsupported or malformed syntax construct; its module evidence was retained.", "path": relative, "location": issue.get("location", issue), "recoverable": True})
                qualified, kind = source_qualified_name(source_root.absolute, absolute_path, name)
                if not qualified:
                    continue
                is_stub = extension == ".pyi"
                file = FileObservation(relative, qualified, kind, stable_id("file", relative), is_stub, is_test_path(relative, name), source_root.relative)
                existing = files.get(relative)
                if existing:
                    if existing.qualified != file.qualified or existing.kind != file.kind:
                        diagnostics.append({"code": "python_conflicting_layout", "severity": "warning", "message": "The same repository file was reached through source roots with conflicting qualified names; the first deterministic observation was retained.", "path": relative, "recoverable": True, "metadata": {"first_qualified_name": existing.qualified, "second_qualified_name": file.qualified, "first_source_root": existing.source_root, "second_source_root": file.source_root}})
                        ambiguous[existing.qualified] = True
                        ambiguous[file.qualified] = True
                    continue
                from_id = module_id(kind, qualified)
                file_imports, file_diagnostics = extract_imports(relative, content, from_id, kind == "package")
                imports.extend(file_imports)
                diagnostics.extend(file_diagnostics)
                files[relative] = file
                sources[file.source_id] = {"id": file.source_id, "path": relative, "symbol": qualified, "kind": "file"}
                for observation in file_imports:
                    sources[observation.source["id"]] = observation.source
                if kind == "package":
                    value = packages.setdefault(qualified, PackageObservation(qualified))
                    value.source_roots.add(file.source_root)
                    value.init_source_roots.add(file.source_root)
                    if value.has_init:
                        conflict = first_conflicting_path(value.paths, relative)
                        if conflict:
                            diagnostics.append({"code": "python_conflicting_layout", "severity": "warning", "message": f"Multiple source files provide the same Python package qualified name; all evidence was retained.", "subject": qualified, "recoverable": True, "metadata": {"kind": "package", "qualified_name": qualified, "first_path": conflict, "second_path": relative}})
                            ambiguous[qualified] = True
                    value.has_init = True
                    value.source_ids.add(file.source_id)
                    value.paths.add(relative)
                    add_file_tags(value.tags, file)
                else:
                    value = modules.setdefault(qualified, ModuleObservation(qualified))
                    conflict = first_conflicting_path(value.paths, relative)
                    if conflict:
                        diagnostics.append({"code": "python_conflicting_layout", "severity": "warning", "message": f"Multiple source files provide the same Python module qualified name; all evidence was retained.", "subject": qualified, "recoverable": True, "metadata": {"kind": "module", "qualified_name": qualified, "first_path": conflict, "second_path": relative}})
                        ambiguous[qualified] = True
                    value.source_ids.add(file.source_id)
                    value.paths.add(relative)
                    add_file_tags(value.tags, file)
                for package_name in parent_packages(qualified):
                    package = packages.setdefault(package_name, PackageObservation(package_name))
                    package.source_roots.add(file.source_root)
                    if not package.has_init:
                        package.source_ids.add(file.source_id)
                        package.paths.add(relative)
                    if kind == "module":
                        add_file_tags(package.tags, file)

    for qualified, value in sorted(packages.items()):
        if not value.init_source_roots:
            continue
        package_ambiguous = len(value.init_source_roots) > 1 or any(root not in value.init_source_roots for root in value.source_roots)
        if package_ambiguous and not ambiguous.get(qualified):
            ambiguous[qualified] = True
            diagnostics.append({"code": "python_conflicting_layout", "severity": "warning", "message": "A regular Python package and namespace content share qualified roots across effective source roots; local resolution was withheld.", "subject": qualified, "recoverable": True, "metadata": {"kind": "package", "qualified_name": qualified, "init_source_roots": sorted(value.init_source_roots), "source_roots": sorted(value.source_roots)}})
    for qualified in sorted(set(packages) & set(modules)):
        ambiguous[qualified] = True
        diagnostics.append({"code": "python_conflicting_layout", "severity": "warning", "message": "A package and module share the same qualified name; both observations were retained with kind-qualified IDs.", "subject": qualified, "recoverable": True, "metadata": {"qualified_name": qualified, "package_id": module_id("package", qualified), "module_id": module_id("module", qualified)}})
    imports.sort(key=lambda value: (value.from_module_id, value.source["id"]))
    sources_list = sorted(sources.values(), key=lambda value: value["id"])
    diagnostics.sort(key=diagnostic_sort_key)
    return Discovery(build_module_observations(project, packages, modules), imports, sources_list, diagnostics, ambiguous)


def join_qualified(left: str, right: str) -> str:
    left, right = left.strip("."), right.strip(".")
    if not left:
        return right
    if not right:
        return left
    return left + "." + right


def package_context(module: dict[str, Any]) -> list[str]:
    qualified = module.get("display_name", "")
    metadata = module.get("metadata", {})
    if metadata.get("qualified_name"):
        qualified = metadata["qualified_name"]
    parts = qualified.split(".")
    if module.get("kind") == "package":
        return [] if not qualified or qualified == "__root__" else parts
    return parts[:-1] if parts and parts != [""] else []


def build_index(modules: list[dict[str, Any]], ambiguous: dict[str, bool]) -> dict[str, Any]:
    by_qualified: dict[str, list[str]] = {}
    by_id: dict[str, dict[str, Any]] = {}
    top_level: set[str] = set()
    values = dict(ambiguous)
    for module in modules:
        by_id[module["id"]] = module
        qualified = module.get("metadata", {}).get("qualified_name") or module["display_name"]
        by_qualified.setdefault(qualified, []).append(module["id"])
        first = qualified.split(".")[0] if qualified else ""
        if first and first != "__root__":
            top_level.add(first)
    for qualified, ids in by_qualified.items():
        ids.sort()
        if len(ids) > 1:
            values[qualified] = True
    return {"by_qualified": by_qualified, "by_id": by_id, "top_level": top_level, "ambiguous": values}


def qualified_path_ambiguous(index: dict[str, Any], qualified: str) -> bool:
    candidate = qualified
    while candidate:
        if index["ambiguous"].get(candidate):
            return True
        candidate = candidate.rsplit(".", 1)[0] if "." in candidate else ""
    return False


def lookup_module(index: dict[str, Any], qualified: str) -> tuple[str, bool]:
    if not qualified:
        return "", False
    ids = index["by_qualified"].get(qualified, [])
    if qualified_path_ambiguous(index, qualified) or len(ids) > 1:
        return "", True
    return (ids[0], False) if len(ids) == 1 else ("", False)


def classify_reference(name: str, index: dict[str, Any], version: str, ambiguous: bool) -> str:
    if ambiguous or name.startswith("."):
        return "unresolved"
    root = name.split(".", 1)[0]
    minimum, maximum = STDLIB_RULES.get(root, (0, 0))
    known_minor = python_version_minor(version)
    stdlib = root in STDLIB_MODULES and (known_minor is None or (not minimum or known_minor >= minimum) and (not maximum or known_minor <= maximum))
    if stdlib:
        return "standard_library"
    if root in index["top_level"]:
        return "unresolved"
    return "external"


def python_version_minor(value: str) -> int | None:
    match = VERSION_VALUE.match(value.strip())
    return int(match.group(1)) if match else None


def condition_state(condition: str, version: str) -> str:
    if not condition:
        return "true"
    state = "true"
    for part in condition.split(" -> "):
        trimmed = part.strip().removesuffix(":")
        lower = trimmed.lower()
        if lower in {"if true", "elif true"}:
            current = "true"
        elif lower in {"if false", "elif false"}:
            current = "false"
        else:
            expression = trimmed.split(" ", 1)[1].strip() if lower.startswith(("if ", "elif ")) else trimmed
            match = VERSION_CONDITION.match(expression)
            minor = python_version_minor(version)
            current = "unknown"
            if match and minor is not None:
                threshold = int(match.group(2))
                operator = match.group(1)
                result = {">=": minor >= threshold, ">": minor > threshold, "<=": minor <= threshold, "<": minor < threshold, "==": minor == threshold, "!=": minor != threshold}[operator]
                current = "true" if result else "false"
        if current == "false":
            return "false"
        if current == "unknown":
            state = "unknown"
    return state


def resolve_import(observation: ImportObservation, index: dict[str, Any], reexports: dict[tuple[str, str], dict[str, Any]] | None) -> dict[str, Any]:
    from_module = index["by_id"].get(observation.from_module_id, {})
    if observation.kind == "import":
        name = observation.module
        target, ambiguous = lookup_module(index, name)
        return {"name": name, "module_id": target, "local": bool(target and not ambiguous), "ambiguous": ambiguous, "resolution": "absolute", "conditional": False, "conditions": []}
    base = observation.module
    if observation.relative_level > 0:
        context = package_context(from_module)
        if not context or observation.relative_level - 1 >= len(context):
            return {"name": "." * observation.relative_level + join_qualified(base, observation.imported_name), "ambiguous": True, "resolution": "relative-parent-outside-root", "local": False, "module_id": "", "conditional": False, "conditions": []}
        context = context[: len(context) - (observation.relative_level - 1)]
        base = join_qualified(".".join(context), base)
    if base:
        _, ambiguous_base = lookup_module(index, base)
        if ambiguous_base:
            name = base if not observation.imported_name or observation.imported_name == "*" else join_qualified(base, observation.imported_name)
            return {"name": name, "ambiguous": True, "resolution": "ambiguous-base", "local": False, "module_id": "", "conditional": False, "conditions": []}
    if not observation.imported_name or observation.imported_name == "*":
        target, ambiguous = lookup_module(index, base)
        return {"name": base, "module_id": target, "local": bool(target and not ambiguous), "ambiguous": ambiguous, "resolution": "base", "conditional": False, "conditions": []}
    child = join_qualified(base, observation.imported_name)
    target, ambiguous = lookup_module(index, child)
    if ambiguous:
        return {"name": child, "ambiguous": True, "resolution": "ambiguous-child", "local": False, "module_id": "", "conditional": False, "conditions": []}
    if target:
        return {"name": child, "module_id": target, "local": True, "ambiguous": False, "resolution": "child", "conditional": False, "conditions": []}
    if reexports is not None:
        value = reexports.get((base, observation.imported_name))
        if value:
            if value.get("ambiguous"):
                return {"name": child, "ambiguous": True, "resolution": "ambiguous-reexport", "local": False, "module_id": "", "conditional": False, "conditions": []}
            return {"name": child, "module_id": value["module_id"], "local": True, "ambiguous": False, "resolution": "reexport", "conditional": value.get("conditional", False), "conditions": list(value.get("conditions", []))}
    if observation.kind == "reexport":
        target, ambiguous = lookup_module(index, base)
        if ambiguous:
            return {"name": base, "ambiguous": True, "resolution": "ambiguous-base", "local": False, "module_id": "", "conditional": False, "conditions": []}
        if target and target != observation.from_module_id:
            return {"name": base, "module_id": target, "local": True, "ambiguous": False, "resolution": "base", "conditional": False, "conditions": []}
    else:
        target, ambiguous = lookup_module(index, base)
        if ambiguous:
            return {"name": child, "ambiguous": True, "resolution": "ambiguous-base", "local": False, "module_id": "", "conditional": False, "conditions": []}
        if target and index["by_id"].get(target, {}).get("kind") == "module":
            return {"name": base, "module_id": target, "local": True, "ambiguous": False, "resolution": "base", "conditional": False, "conditions": []}
    return {"name": child, "ambiguous": False, "resolution": "unresolved", "local": False, "module_id": "", "conditional": False, "conditions": []}


def collect_reexports(project: Project, discovery: Discovery, index: dict[str, Any]) -> dict[tuple[str, str], dict[str, Any]]:
    result: dict[tuple[str, str], dict[str, Any]] = {}
    for observation in discovery.imports:
        if observation.kind != "reexport" or not observation.imported_name or observation.imported_name == "*":
            continue
        if condition_state(observation.condition, project.python_version) == "false":
            continue
        from_module = index["by_id"].get(observation.from_module_id, {})
        if from_module.get("kind") != "package" or from_module.get("display_name") == "__root__":
            continue
        resolution = resolve_import(observation, index, None)
        if not resolution.get("local"):
            continue
        name = observation.alias or observation.imported_name
        key = (from_module.get("display_name", ""), name)
        value = result.setdefault(key, {"module_id": resolution["module_id"], "ambiguous": False, "conditional": False, "conditions": [], "unconditional": False})
        if value["module_id"] != resolution["module_id"]:
            value["ambiguous"] = True
        if observation.conditional and condition_state(observation.condition, project.python_version) != "true":
            if not value["unconditional"]:
                value["conditional"] = True
                if observation.condition not in value["conditions"]:
                    value["conditions"].append(observation.condition)
        else:
            value["unconditional"] = True
            value["conditional"] = False
            value["conditions"] = []
    return result


def append_unique(values: list[Any], value: Any) -> list[Any]:
    if value not in values:
        values.append(value)
    return values


def merge_metadata(target: dict[str, Any], values: dict[str, Any] | None) -> None:
    if not values:
        return
    for key, value in values.items():
        if isinstance(value, list):
            current = target.setdefault(key, [])
            for item in value:
                if item not in current:
                    current.append(item)
            current.sort(key=lambda item: json_key(item))
        elif isinstance(value, bool):
            if key not in target or value:
                target[key] = value
        elif isinstance(value, str):
            if key not in target:
                target[key] = value
            elif key == "condition_evaluation" and target[key] != value:
                target[key] = "unknown"
        elif key not in target:
            target[key] = value


def import_metadata(observation: ImportObservation, scope: str, resolution: dict[str, Any]) -> dict[str, Any]:
    metadata: dict[str, Any] = {
        "target_scope": scope,
        "import_paths": [observation.spelling],
        "import_kinds": [observation.kind],
        "resolution_kinds": [resolution["resolution"]],
        "relative_levels": [observation.relative_level],
        "import_names": [],
    }
    if observation.imported_name:
        metadata["import_names"] = [observation.imported_name]
    if observation.alias:
        metadata["aliases"] = [observation.alias]
    if observation.relative_level > 0:
        metadata["relative"] = True
    if observation.kind == "reexport" or resolution["resolution"] == "reexport":
        metadata["reexport"] = True
    conditions: list[str] = []
    if observation.conditional:
        conditions.append(observation.condition)
    if resolution.get("conditional"):
        conditions.extend(resolution.get("conditions", []))
    if conditions:
        metadata["conditional"] = True
        metadata["conditions"] = sorted(set(conditions))
    return metadata


def reference_metadata(observation: ImportObservation, scope: str, resolution: dict[str, Any], version: str) -> dict[str, Any]:
    metadata = import_metadata(observation, scope, resolution)
    metadata["reference_scope"] = scope
    if observation.conditional:
        metadata["condition_evaluation"] = condition_state(observation.condition, version)
    if resolution.get("ambiguous"):
        metadata["ambiguous"] = True
    if version:
        metadata["python_version"] = version
    return metadata


def confidence(scope: str, conditional: bool) -> dict[str, Any]:
    if scope == "dynamic":
        return {"basis": "dynamic", "score": 0.2}
    if scope == "unresolved":
        return {"basis": "unresolved", "score": 0.2}
    if conditional:
        return {"basis": "inferred", "score": 0.5}
    return {"basis": "resolved", "score": 1}


def merge_confidence(left: dict[str, Any] | None, right: dict[str, Any] | None) -> dict[str, Any] | None:
    if left is None:
        return right
    if right is None:
        return left
    return right if right.get("score", 0) < left.get("score", 0) else left


def import_diagnostic(observation: ImportObservation, code: str, message: str, severity: str, recoverable: bool, metadata: dict[str, Any]) -> dict[str, Any]:
    metadata = dict(metadata)
    metadata["source_reference_id"] = observation.source["id"]
    return {"code": code, "severity": severity, "message": message, "path": observation.source["path"], "location": observation.source.get("start"), "recoverable": recoverable, "metadata": metadata}


def add_dynamic_observation(project: Project, observation: ImportObservation, relationships: dict[str, dict[str, Any]], references: dict[str, dict[str, Any]], diagnostics: list[dict[str, Any]], state: str) -> None:
    name = observation.dynamic_target or "<dynamic>"
    reference_id = stable_id("reference", "python", "dynamic", name)
    reference = references.setdefault(reference_id, {"id": reference_id, "name": name, "scope": "dynamic", "language": "python", "metadata": {}})
    metadata: dict[str, Any] = {"target_scope": "dynamic", "dynamic_functions": [observation.dynamic_function], "dynamic_targets": [], "import_kinds": ["dynamic"]}
    if observation.dynamic_target:
        metadata["dynamic_targets"] = [observation.dynamic_target]
    if observation.conditional:
        metadata.update({"conditional": True, "conditions": [observation.condition], "condition_evaluation": state})
    if project.python_version:
        metadata["python_version"] = project.python_version
    merge_metadata(reference["metadata"], metadata)
    relationship_id = stable_id("relationship", observation.from_module_id, "dynamic", reference_id)
    relationship = relationships.setdefault(relationship_id, {"id": relationship_id, "type": "depends_on", "from_module_id": observation.from_module_id, "to_reference_id": reference_id, "source_reference_ids": [], "confidence": confidence("dynamic", False), "metadata": {}})
    append_unique(relationship["source_reference_ids"], observation.source["id"])
    merge_metadata(relationship["metadata"], metadata)
    relationship["confidence"] = merge_confidence(relationship.get("confidence"), confidence("dynamic", False))
    diagnostics.append(import_diagnostic(observation, "python_dynamic_import", f"Python dynamic import {quoted(name)} through {observation.dynamic_function} could not be resolved statically; it was retained as a dynamic reference.", "warning", True, metadata))
    if observation.conditional and state == "unknown":
        diagnostics.append(import_diagnostic(observation, "python_conditional_import", "Python dynamic import condition could not be proven for the selected view; the observation is partial.", "warning", True, {"condition": observation.condition, "target_scope": "dynamic"}))


def build_import_observations(project: Project, discovery: Discovery) -> tuple[list[dict[str, Any]], list[dict[str, Any]], list[dict[str, Any]]]:
    index = build_index(discovery.modules, discovery.ambiguous_names)
    reexports = collect_reexports(project, discovery, index)
    relationships: dict[str, dict[str, Any]] = {}
    references: dict[str, dict[str, Any]] = {}
    diagnostics: list[dict[str, Any]] = []
    for observation in discovery.imports:
        state = condition_state(observation.condition, project.python_version)
        if state == "false":
            diagnostics.append(import_diagnostic(observation, "python_conditional_import_excluded", "Python import was excluded because its condition is false for the selected Python version.", "info", False, {"condition": observation.condition, "condition_evaluation": "false"}))
            continue
        if observation.kind == "dynamic":
            add_dynamic_observation(project, observation, relationships, references, diagnostics, state)
            continue
        resolution = resolve_import(observation, index, reexports)
        if resolution.get("local") and resolution.get("module_id") == observation.from_module_id:
            continue
        scope = "local" if resolution.get("local") else classify_reference(resolution["name"], index, project.python_version, bool(resolution.get("ambiguous") or observation.relative_level > 0))
        metadata = import_metadata(observation, scope, resolution)
        conditional = bool(resolution.get("conditional") or (observation.conditional and state != "true"))
        if conditional:
            metadata["condition_evaluation"] = "unknown"
        elif observation.conditional:
            metadata["condition_evaluation"] = state
        target_id = resolution.get("module_id", "") if resolution.get("local") else ""
        reference_id = ""
        if not resolution.get("local"):
            reference_id = stable_id("reference", "python", scope, resolution["name"])
            reference = references.setdefault(reference_id, {"id": reference_id, "name": resolution["name"], "scope": scope, "language": "python", "metadata": {}})
            merge_metadata(reference["metadata"], reference_metadata(observation, scope, resolution, project.python_version))
        relationship_id = stable_id("relationship", observation.from_module_id, scope, target_id, reference_id)
        relationship = relationships.setdefault(relationship_id, {"id": relationship_id, "type": "depends_on", "from_module_id": observation.from_module_id, "source_reference_ids": [], "confidence": confidence(scope, conditional), "metadata": {}})
        if target_id:
            relationship["to_module_id"] = target_id
        if reference_id:
            relationship["to_reference_id"] = reference_id
        append_unique(relationship["source_reference_ids"], observation.source["id"])
        merge_metadata(relationship["metadata"], metadata)
        relationship["confidence"] = merge_confidence(relationship.get("confidence"), confidence(scope, conditional))
        if scope == "unresolved":
            reason = "has multiple possible project-local targets" if resolution.get("ambiguous") else "could not be resolved to a proven project-local module"
            diagnostics.append(import_diagnostic(observation, "python_unresolved_import", f"Python import {quoted(observation.spelling)} {reason}; it was retained as an unresolved reference.", "warning", True, {"target_scope": scope, "reference_id": reference_id, "resolution": resolution["resolution"]}))
        if conditional and state == "unknown":
            diagnostics.append(import_diagnostic(observation, "python_conditional_import", "Python import condition could not be proven for the selected view; the observation is partial.", "warning", True, {"condition": observation.condition, "target_scope": scope}))
    for value in relationships.values():
        value["source_reference_ids"].sort()
        merge_metadata(value.get("metadata", {}), None)
    for value in references.values():
        merge_metadata(value.get("metadata", {}), None)
    return sorted(relationships.values(), key=lambda value: value["id"]), sorted(references.values(), key=lambda value: value["id"]), diagnostics


class AnalysisFailure(Exception):
    def __init__(self, code: str, message: str, details: dict[str, Any] | None = None) -> None:
        super().__init__(message)
        self.code = code
        self.message = message
        self.details = details or {}


def analyze(project_root: str, options: dict[str, Any]) -> dict[str, Any]:
    project = resolve_project(project_root, options)
    discovery = discover(project, options)
    relationships, references, import_diagnostics = build_import_observations(project, discovery)
    diagnostics = list(discovery.diagnostics) + import_diagnostics
    if not discovery.modules:
        diagnostics.append({"code": "python_no_modules", "severity": "warning", "message": "No eligible Python packages or modules were found in the selected source roots.", "recoverable": True})
    diagnostics.sort(key=diagnostic_sort_key)
    status = "partial" if any(value.get("recoverable") for value in diagnostics) else "complete"
    modules = discovery.modules
    result = {
        "run_id": "",
        "status": status,
        "analyzer": {"id": MANIFEST["id"], "version": MANIFEST["version"], "language": MANIFEST["language"], "api_version": MANIFEST["api_version"]},
        "project": {"root_label": os.path.basename(project.root), "boundary": project.boundary, "module_root": project.source_roots[0].relative if project.source_roots else "."},
        "options_fingerprint": str(options.get("__fingerprint", "")),
        "modules": modules,
        "relationships": relationships,
        "references": references,
        "source_references": discovery.source_references,
        "diagnostics": diagnostics,
        "summary": {"module_count": len(modules), "relationship_count": len(relationships), "reference_count": len(references), "source_reference_count": len(discovery.source_references), "diagnostic_count": len(diagnostics)},
    }
    return result


def detect(project_root: str) -> dict[str, Any]:
    root = os.path.normpath(os.path.abspath(project_root))
    if not os.path.isdir(root):
        raise AnalysisFailure("unreadable_project", "Python project root does not exist or is not a directory")
    matched: list[str] = []
    confidence_value = 0.0
    boundary = ""
    for marker in MANIFEST["detection_markers"]:
        if safe_project_file(root, marker["value"]):
            matched.append(marker["value"])
            if not boundary:
                boundary = marker["value"]
                confidence_value = marker["weight"]
    return {"analyzer_id": ANALYZER_ID, "confidence": confidence_value, "matched_markers": matched, "boundary_hint": boundary, "reason": "Python project marker detection"}


def analyze_request(project_root: str, options: dict[str, Any], fingerprint: str) -> dict[str, Any]:
    values = dict(options)
    values["__fingerprint"] = fingerprint
    return analyze(project_root, values)
