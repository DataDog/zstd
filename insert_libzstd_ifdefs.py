#!/usr/bin/env python
import sys
import glob

FLAG="USE_EXTERNAL_ZSTD"

def insert_ifdefs(file):

    with open(file, "r") as fd:
        content=fd.read()

    with open(file, "w") as fd:
        fd.seek(0)
        fd.write(f"#ifndef {FLAG}\n")
        fd.write(content)
        if file == "zstd.h":
            fd.write("\n".join([
            f"#else /* {FLAG} */",
            "#undef ZSTD_STATIC_LINKING_ONLY",
            "#include_next <zstd.h>",
            f"#endif /* {FLAG} */"]))
        else:
            fd.write(f"\n#endif /* {FLAG} */\n")


if __name__ == "__main__":
    for file in glob.glob("*.c") + glob.glob("*.h"):
        insert_ifdefs(file)
