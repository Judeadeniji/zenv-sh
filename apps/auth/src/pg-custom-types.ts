import { customType } from "drizzle-orm/pg-core";


export const bytea = customType<{ data: Buffer; }>({
  dataType() {
    return 'bytea';
  },
  toDriver(value: Buffer) {
    return value;
  },
  fromDriver(value: unknown) {
    // Some drivers return the value as a Buffer already, 
    // but this ensures consistency.
    if (Buffer.isBuffer(value)) return value;
    return Buffer.from(value as ArrayLike<number>);
  },
});
