#!/bin/bash

# Run this after updating vendored zstd sources.

for filename in *.h *c; do
    if ! grep -F "#ifndef USE_LIBZSTD" "$filename" >/dev/null; then
        if [[ $filename == "zstd.h" ]]; then
            sed -i "$filename" \
                -e '1 i\#ifndef USE_LIBZSTD' \
                -e '$ a\#else /* USE_LIBZSTD */\n#undef ZSTD_STATIC_LINKING_ONLY\n#include_next <zstd.h>\n#endif /* USE_LIBZSTD */'
        else
            sed -i "$filename" \
                -e '1 i\#ifndef USE_LIBZSTD' \
                -e '$ a\#endif /* USE_LIBZSTD */'
        fi
    fi
done
