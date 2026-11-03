export const REPOSITORY_CONFIG_TOOLTIPS = {
  chunking:
    "A chunking algorithm splits data into smaller pieces, or “chunks,” based on specific rules (e.g., fixed size or content-defined). This enables efficient deduplication by identifying and eliminating redundant data. Plakar currently uses FastCDC, a fast and content-aware chunking algorithm.",
  compression:
    "A compression algorithm reduces the size of data by encoding patterns and redundancies, helping save storage space and speed up transfers. Different algorithms offer trade-offs between speed, compression ratio, and resource usage. Plakar currently uses LZ4, a high-speed compression algorithm optimized for fast storage and retrieval.",
  encryption:
    "An encryption algorithm protects data by transforming it into an unreadable format, ensuring confidentiality and integrity. In encrypted repositories, the only way to access the data is with the correct passphrase — without it, the data remains completely inaccessible.",
};
