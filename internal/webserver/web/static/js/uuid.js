function getUuid() {
  // Native UUID, not available in insecure environment
  if (typeof crypto !== "undefined" && crypto.randomUUID) {
    return crypto.randomUUID();
  }

  // CSPRNG-backed fallback
  if (typeof crypto !== "undefined" && crypto.getRandomValues) {
    const bytes = new Uint8Array(16);
    crypto.getRandomValues(bytes);

    // RFC 4122 compliance
    bytes[6] = (bytes[6] & 0x0f) | 0x40; // version 4
    bytes[8] = (bytes[8] & 0x3f) | 0x80; // variant 10

    return [...bytes]
      .map((b, i) =>
        (i === 4 || i === 6 || i === 8 || i === 10 ? "-" : "") +
        b.toString(16).padStart(2, "0")
      )
      .join("");
  }

  // If unavailable, Math.random (not cryptographically secure)
  let uuid = "", i;
  for (i = 0; i < 36; i++) {
    if (i === 8 || i === 13 || i === 18 || i === 23) {
      uuid += "-";
    } else if (i === 14) {
      uuid += "4";
    } else {
      const r = Math.random() * 16 | 0;
      uuid += (i === 19 ? (r & 0x3) | 0x8 : r).toString(16);
    }
  }
  return uuid;
}
