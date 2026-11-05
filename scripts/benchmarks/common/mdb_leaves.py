"""Enumerate every leaf telemetry parameter from XTCE MDB files.

Aggregates and arrays are expanded recursively. Multi-dimensional arrays
expand per dimension. Paths use Yamcs naming, e.g.
``/features/DemoPowerMatrix[1][2]`` or ``/features/DemoBundle.attitude.yaw``.
"""

from __future__ import annotations

import itertools
import xml.etree.ElementTree as ET
from pathlib import Path

AGGREGATE = "AggregateParameterType"
ARRAY = "ArrayParameterType"


def _local(tag: str) -> str:
    return tag.split("}", 1)[1] if "}" in tag else tag


def _iter_tag(elem, name):
    return [c for c in elem.iter() if _local(c.tag) == name]


def _fixed(dim, which):
    for c in dim.iter():
        if _local(c.tag) == which:
            for v in c.iter():
                if _local(v.tag) == "FixedValue" and v.text:
                    return int(v.text.strip())
    raise ValueError(f"Dimension without FixedValue for {which}")


def _array_ranges(type_elem):
    return [
        range(_fixed(dim, "StartingIndex"), _fixed(dim, "EndingIndex") + 1)
        for dim in _iter_tag(type_elem, "Dimension")
    ]


def _load_types(root):
    return {
        t.get("name"): t
        for t in root.iter()
        if _local(t.tag) in (AGGREGATE, ARRAY) and t.get("name")
    }


def _expand(type_name, types, path):
    elem = types.get(type_name)
    if elem is None:
        yield path
        return
    if _local(elem.tag) == AGGREGATE:
        for member in _iter_tag(elem, "Member"):
            yield from _expand(member.get("typeRef"), types, f"{path}.{member.get('name')}")
    else:
        base = elem.get("arrayTypeRef")
        for idx in itertools.product(*_array_ranges(elem)):
            yield from _expand(base, types, path + "".join(f"[{i}]" for i in idx))


def leaves_for_xtce(xml_path: Path):
    """Return sorted Yamcs leaf paths for every top-level Parameter in one XTCE file."""
    root = ET.parse(xml_path).getroot()
    space = next(
        (e.get("name") for e in root.iter() if _local(e.tag) == "SpaceSystem" and e.get("name")),
        xml_path.stem,
    )
    types = _load_types(root)
    out = []
    for p in root.iter():
        if _local(p.tag) != "Parameter" or not p.get("name") or not p.get("parameterTypeRef"):
            continue
        out.extend(_expand(p.get("parameterTypeRef"), types, f"/{space}/{p.get('name')}"))
    return sorted(set(out))


def enumerate_mdb(mdb_dir: Path):
    """Return {xml filename: sorted leaf paths} for every XTCE file in mdb_dir."""
    return {x.name: leaves_for_xtce(x) for x in sorted(Path(mdb_dir).glob("*.xml"))}


def main(argv=None):
    """Regenerate ``grouped_parameters`` in workload.json from an MDB directory."""
    import argparse
    import json

    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--mdb", required=True, help="directory containing XTCE *.xml files")
    parser.add_argument("--workload", default=str(Path(__file__).resolve().parent.parent / "workload.json"))
    parser.add_argument("--qualified-results", help="live group-work result defining the eligible scalar pool")
    args = parser.parse_args(argv)

    leaves = []
    for name, paths in enumerate_mdb(Path(args.mdb)).items():
        leaves.extend(paths)
    leaves = sorted(set(leaves))

    workload_path = Path(args.workload)
    with open(workload_path, encoding="utf-8") as fh:
        workload = json.load(fh)
    workload["grouped_parameters"] = leaves
    if args.qualified_results:
        with open(args.qualified_results, encoding="utf-8") as fh:
            qualified = json.load(fh)
        eligible = qualified["parameters"]
        unavailable = qualified["unavailable_parameters"]
        if set(eligible).intersection(unavailable) or sorted(set(eligible + unavailable)) != leaves:
            raise ValueError("Live qualification does not cover exactly this MDB's declared leaves")
        workload["parameters"] = eligible
        workload["unavailable_parameters"] = unavailable
    with open(workload_path, "w", encoding="utf-8") as fh:
        json.dump(workload, fh, indent=2)
        fh.write("\n")
    print(f"wrote {len(leaves)} grouped_parameters to {workload_path}")


if __name__ == "__main__":
    main()
