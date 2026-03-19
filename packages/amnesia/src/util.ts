/**
 * Convert a Uint8Array to an ArrayBuffer suitable for Web Crypto API.
 * Fixes TS 5.9+ strict typing where Uint8Array.buffer returns ArrayBufferLike
 * which is not assignable to BufferSource.
 */
export function toBuffer(data: Uint8Array): ArrayBuffer {
  return data.buffer.slice(
    data.byteOffset,
    data.byteOffset + data.byteLength,
  ) as ArrayBuffer;
}
