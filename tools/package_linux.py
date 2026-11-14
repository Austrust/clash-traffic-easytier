"""Normalize Linux package permissions after a Windows MyGo cross-build."""
import io
from pathlib import Path
import tarfile


def normalize_tar(data):
    output = io.BytesIO()
    with tarfile.open(fileobj=io.BytesIO(data), mode="r:gz") as source:
        with tarfile.open(fileobj=output, mode="w:gz") as target:
            for member in source:
                if member.isfile():
                    member.mode = 0o755 if Path(member.name).name == "clash-traffic" else 0o644
                elif member.isdir():
                    member.mode = 0o755
                target.addfile(member, source.extractfile(member) if member.isfile() else None)
    return output.getvalue()


def normalize_deb(data):
    assert data.startswith(b"!<arch>\n"), "Invalid Debian ar archive"
    output = bytearray(data[:8])
    offset = 8
    while offset < len(data):
        header = data[offset:offset + 60]
        assert len(header) == 60 and header[58:] == b"`\n"
        size = int(header[48:58])
        body = data[offset + 60:offset + 60 + size]
        if header[:16].decode().strip().rstrip("/") == "data.tar.gz":
            body = normalize_tar(body)
        output.extend(header[:48] + str(len(body)).encode().ljust(10) + header[58:])
        output.extend(body)
        if len(body) % 2:
            output.extend(b"\n")
        offset += 60 + size + size % 2
    return bytes(output)


if __name__ == "__main__":
    root = Path(__file__).resolve().parents[1]
    for arch in ("amd64", "arm64"):
        directory = root / "build" / ("linux-" + arch)
        for path in directory.glob("*.tar.gz"):
            path.write_bytes(normalize_tar(path.read_bytes()))
            print("Normalized", path.name)
        for path in directory.glob("*.deb"):
            path.write_bytes(normalize_deb(path.read_bytes()))
            print("Normalized", path.name)