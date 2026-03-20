/**
 * Convert a Uint8Array to an ArrayBuffer suitable for Web Crypto API.
 * Skips the copy when the view already spans the entire backing buffer.
 */
export function toBuffer(data: Uint8Array): ArrayBuffer {
  if (data.byteOffset === 0 && data.byteLength === data.buffer.byteLength) {
    return data.buffer as ArrayBuffer;
  }
  return data.buffer.slice(
    data.byteOffset,
    data.byteOffset + data.byteLength,
  ) as ArrayBuffer;
}
