/*
    zstd - standard compression library
    Header File for static linking only
    Copyright (C) 2014-2016, Yann Collet.

    BSD 2-Clause License (http://www.opensource.org/licenses/bsd-license.php)

    Redistribution and use in source and binary forms, with or without
    modification, are permitted provided that the following conditions are
    met:
    * Redistributions of source code must retain the above copyright
    notice, this list of conditions and the following disclaimer.
    * Redistributions in binary form must reproduce the above
    copyright notice, this list of conditions and the following disclaimer
    in the documentation and/or other materials provided with the
    distribution.
    THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
    "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
    LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
    A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
    OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
    SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
    LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
    DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
    THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
    (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
    OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

    You can contact the author at :
    - zstd source repository : https://github.com/Cyan4973/zstd
*/
#ifndef ZSTD_0_5_X_STATIC_H
#define ZSTD_0_5_X_STATIC_H

/* The objects defined into this file shall be considered experimental.
 * They are not considered stable, as their prototype may change in the future.
 * You can use them for tests, provide feedback, or if you can endure risks of future changes.
 */

#if defined (__cplusplus)
extern "C" {
#endif

/*-*************************************
*  Dependencies
***************************************/
#include "zstd.h"
#include "mem.h"


/*-*************************************
*  Types
***************************************/
#define ZSTD_0_5_X_WINDOWLOG_MAX 26
#define ZSTD_0_5_X_WINDOWLOG_MIN 18
#define ZSTD_0_5_X_WINDOWLOG_ABSOLUTEMIN 11
#define ZSTD_0_5_X_CONTENTLOG_MAX (ZSTD_0_5_X_WINDOWLOG_MAX+1)
#define ZSTD_0_5_X_CONTENTLOG_MIN 4
#define ZSTD_0_5_X_HASHLOG_MAX 28
#define ZSTD_0_5_X_HASHLOG_MIN 4
#define ZSTD_0_5_X_SEARCHLOG_MAX (ZSTD_0_5_X_CONTENTLOG_MAX-1)
#define ZSTD_0_5_X_SEARCHLOG_MIN 1
#define ZSTD_0_5_X_SEARCHLENGTH_MAX 7
#define ZSTD_0_5_X_SEARCHLENGTH_MIN 4

/** from faster to stronger */
typedef enum { ZSTD_0_5_X_fast, ZSTD_0_5_X_greedy, ZSTD_0_5_X_lazy, ZSTD_0_5_X_lazy2, ZSTD_0_5_X_btlazy2 } ZSTD_0_5_X_strategy;

typedef struct
{
    uint64_t srcSize;       /* optional : tells how much bytes are present in the frame. Use 0 if not known. */
    uint32_t windowLog;     /* largest match distance : larger == more compression, more memory needed during decompression */
    uint32_t contentLog;    /* full search segment : larger == more compression, slower, more memory (useless for fast) */
    uint32_t hashLog;       /* dispatch table : larger == more memory, faster */
    uint32_t searchLog;     /* nb of searches : larger == more compression, slower */
    uint32_t searchLength;  /* size of matches : larger == faster decompression, sometimes less compression */
    ZSTD_0_5_X_strategy strategy;
} ZSTD_0_5_X_parameters;


/* *************************************
*  Advanced functions
***************************************/
#define ZSTD_0_5_X_MAX_CLEVEL 20
ZSTDLIB_API unsigned ZSTD_0_5_X_maxCLevel (void);

/*! ZSTD_0_5_X_getParams() :
*   @return ZSTD_0_5_X_parameters structure for a selected compression level and srcSize.
*   `srcSizeHint` value is optional, select 0 if not known */
ZSTDLIB_API ZSTD_0_5_X_parameters ZSTD_0_5_X_getParams(int compressionLevel, U64 srcSizeHint);

/*! ZSTD_0_5_X_validateParams() :
*   correct params value to remain within authorized range */
ZSTDLIB_API void ZSTD_0_5_X_validateParams(ZSTD_0_5_X_parameters* params);

/*! ZSTD_0_5_X_compress_advanced() :
*   Same as ZSTD_0_5_X_compress_usingDict(), with fine-tune control of each compression parameter */
ZSTDLIB_API size_t ZSTD_0_5_X_compress_advanced (ZSTD_0_5_X_CCtx* ctx,
                                           void* dst, size_t dstCapacity,
                                     const void* src, size_t srcSize,
                                     const void* dict,size_t dictSize,
                                           ZSTD_0_5_X_parameters params);

/*! ZSTD_0_5_X_compress_usingPreparedDCtx() :
*   Same as ZSTD_0_5_X_compress_usingDict, but using a reference context `preparedCCtx`, where dictionary has been loaded.
*   It avoids reloading the dictionary each time.
*   `preparedCCtx` must have been properly initialized using ZSTD_0_5_X_compressBegin_usingDict() or ZSTD_0_5_X_compressBegin_advanced().
*   Requires 2 contexts : 1 for reference, which will not be modified, and 1 to run the compression operation */
ZSTDLIB_API size_t ZSTD_0_5_X_compress_usingPreparedCCtx(
                                           ZSTD_0_5_X_CCtx* cctx, const ZSTD_0_5_X_CCtx* preparedCCtx,
                                           void* dst, size_t dstCapacity,
                                     const void* src, size_t srcSize);

/*- Advanced Decompression functions -*/

/*! ZSTD_0_5_X_decompress_usingPreparedDCtx() :
*   Same as ZSTD_0_5_X_decompress_usingDict, but using a reference context `preparedDCtx`, where dictionary has been loaded.
*   It avoids reloading the dictionary each time.
*   `preparedDCtx` must have been properly initialized using ZSTD_0_5_X_decompressBegin_usingDict().
*   Requires 2 contexts : 1 for reference, which will not be modified, and 1 to run the decompression operation */
ZSTDLIB_API size_t ZSTD_0_5_X_decompress_usingPreparedDCtx(
                                             ZSTD_0_5_X_DCtx* dctx, const ZSTD_0_5_X_DCtx* preparedDCtx,
                                             void* dst, size_t dstCapacity,
                                       const void* src, size_t srcSize);


/* **************************************
*  Streaming functions (direct mode)
****************************************/
ZSTDLIB_API size_t ZSTD_0_5_X_compressBegin(ZSTD_0_5_X_CCtx* cctx, int compressionLevel);
ZSTDLIB_API size_t ZSTD_0_5_X_compressBegin_usingDict(ZSTD_0_5_X_CCtx* cctx, const void* dict,size_t dictSize, int compressionLevel);
ZSTDLIB_API size_t ZSTD_0_5_X_compressBegin_advanced(ZSTD_0_5_X_CCtx* cctx, const void* dict,size_t dictSize, ZSTD_0_5_X_parameters params);
ZSTDLIB_API size_t ZSTD_0_5_X_copyCCtx(ZSTD_0_5_X_CCtx* cctx, const ZSTD_0_5_X_CCtx* preparedCCtx);

ZSTDLIB_API size_t ZSTD_0_5_X_compressContinue(ZSTD_0_5_X_CCtx* cctx, void* dst, size_t dstCapacity, const void* src, size_t srcSize);
ZSTDLIB_API size_t ZSTD_0_5_X_compressEnd(ZSTD_0_5_X_CCtx* cctx, void* dst, size_t dstCapacity);

/*
  Streaming compression, synchronous mode (bufferless)

  A ZSTD_0_5_X_CCtx object is required to track streaming operations.
  Use ZSTD_0_5_X_createCCtx() / ZSTD_0_5_X_freeCCtx() to manage it.
  ZSTD_0_5_X_CCtx object can be re-used multiple times within successive compression operations.

  Start by initializing a context.
  Use ZSTD_0_5_X_compressBegin(), or ZSTD_0_5_X_compressBegin_usingDict() for dictionary compression,
  or ZSTD_0_5_X_compressBegin_advanced(), for finer parameter control.
  It's also possible to duplicate a reference context which has been initialized, using ZSTD_0_5_X_copyCCtx()

  Then, consume your input using ZSTD_0_5_X_compressContinue().
  The interface is synchronous, so all input will be consumed and produce a compressed output.
  You must ensure there is enough space in destination buffer to store compressed data under worst case scenario.
  Worst case evaluation is provided by ZSTD_0_5_X_compressBound().

  Finish a frame with ZSTD_0_5_X_compressEnd(), which will write the epilogue.
  Without the epilogue, frames will be considered incomplete by decoder.

  You can then reuse ZSTD_0_5_X_CCtx to compress some new frame.
*/


ZSTDLIB_API size_t ZSTD_0_5_X_decompressBegin(ZSTD_0_5_X_DCtx* dctx);
ZSTDLIB_API size_t ZSTD_0_5_X_decompressBegin_usingDict(ZSTD_0_5_X_DCtx* dctx, const void* dict, size_t dictSize);
ZSTDLIB_API void   ZSTD_0_5_X_copyDCtx(ZSTD_0_5_X_DCtx* dctx, const ZSTD_0_5_X_DCtx* preparedDCtx);

ZSTDLIB_API size_t ZSTD_0_5_X_getFrameParams(ZSTD_0_5_X_parameters* params, const void* src, size_t srcSize);

ZSTDLIB_API size_t ZSTD_0_5_X_nextSrcSizeToDecompress(ZSTD_0_5_X_DCtx* dctx);
ZSTDLIB_API size_t ZSTD_0_5_X_decompressContinue(ZSTD_0_5_X_DCtx* dctx, void* dst, size_t dstCapacity, const void* src, size_t srcSize);

/*
  Streaming decompression, direct mode (bufferless)

  A ZSTD_0_5_X_DCtx object is required to track streaming operations.
  Use ZSTD_0_5_X_createDCtx() / ZSTD_0_5_X_freeDCtx() to manage it.
  A ZSTD_0_5_X_DCtx object can be re-used multiple times.

  First typical operation is to retrieve frame parameters, using ZSTD_0_5_X_getFrameParams().
  This operation is independent, and just needs enough input data to properly decode the frame header.
  Objective is to retrieve *params.windowlog, to know minimum amount of memory required during decoding.
  Result : 0 when successful, it means the ZSTD_0_5_X_parameters structure has been filled.
           >0 : means there is not enough data into src. Provides the expected size to successfully decode header.
           errorCode, which can be tested using ZSTD_0_5_X_isError()

  Start decompression, with ZSTD_0_5_X_decompressBegin() or ZSTD_0_5_X_decompressBegin_usingDict()
  Alternatively, you can copy a prepared context, using ZSTD_0_5_X_copyDCtx()

  Then use ZSTD_0_5_X_nextSrcSizeToDecompress() and ZSTD_0_5_X_decompressContinue() alternatively.
  ZSTD_0_5_X_nextSrcSizeToDecompress() tells how much bytes to provide as 'srcSize' to ZSTD_0_5_X_decompressContinue().
  ZSTD_0_5_X_decompressContinue() requires this exact amount of bytes, or it will fail.
  ZSTD_0_5_X_decompressContinue() needs previous data blocks during decompression, up to (1 << windowlog).
  They should preferably be located contiguously, prior to current block. Alternatively, a round buffer is also possible.

  @result of ZSTD_0_5_X_decompressContinue() is the number of bytes regenerated within 'dst'.
  It can be zero, which is not an error; it just means ZSTD_0_5_X_decompressContinue() has decoded some header.

  A frame is fully decoded when ZSTD_0_5_X_nextSrcSizeToDecompress() returns zero.
  Context can then be reset to start a new decompression.
*/


/* **************************************
*  Block functions
****************************************/
/*! Block functions produce and decode raw zstd blocks, without frame metadata.
    User will have to save and regenerate necessary information to regenerate data, such as block sizes.

    A few rules to respect :
    - Uncompressed block size must be <= 128 KB
    - Compressing or decompressing requires a context structure
      + Use ZSTD_0_5_X_createCCtx() and ZSTD_0_5_X_createDCtx()
    - It is necessary to init context before starting
      + compression : ZSTD_0_5_X_compressBegin()
      + decompression : ZSTD_0_5_X_decompressBegin()
      + variants _usingDict() are also allowed
      + copyCCtx() and copyDCtx() work too
    - When a block is considered not compressible enough, ZSTD_0_5_X_compressBlock() result will be zero.
      In which case, nothing is produced into `dst`.
      + User must test for such outcome and deal directly with uncompressed data
      + ZSTD_0_5_X_decompressBlock() doesn't accept uncompressed data as input !!
*/

size_t ZSTD_0_5_X_compressBlock  (ZSTD_0_5_X_CCtx* cctx, void* dst, size_t dstCapacity, const void* src, size_t srcSize);
size_t ZSTD_0_5_X_decompressBlock(ZSTD_0_5_X_DCtx* dctx, void* dst, size_t dstCapacity, const void* src, size_t srcSize);


/* *************************************
*  Error management
***************************************/
#include "error_public.h"
/*! ZSTD_0_5_X_getErrorCode() :
    convert a `size_t` function result into a `ZSTD_0_5_X_error_code` enum type,
    which can be used to compare directly with enum list within "error_public.h" */
ZSTD_0_5_X_ErrorCode ZSTD_0_5_X_getError(size_t code);


#if defined (__cplusplus)
}
#endif

#endif  /* ZSTD_0_5_X_STATIC_H */
