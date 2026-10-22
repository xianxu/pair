// Private delivery is ordered facts, not another sketch authority machine.
export const SKETCH_BOOTSTRAP_BYTES = 36 * 1024 * 1024;
export const SKETCH_BATCH_BYTES = 64 * 1024;
export const SKETCH_TRANSFER_BYTES = 32 * 1024;
const encoder = new TextEncoder();
const safe = (n) => Number.isSafeInteger(n) && n >= 0;
const need = (ok, why) => {
  if (!ok) throw Error(`sketch delivery: ${why}`);
};
const identity = (value) =>
  typeof value === 'string' && value.length > 0 && encoder.encode(value).length <= 128;
function encoded(value, limit) {
  const json = JSON.stringify(value);
  need(encoder.encode(json).length <= limit, 'byte limit');
  return json;
}

export function createSketchDelivery({ installBootstrap, applyBatch }) {
  let epoch = null,
    seq = 0,
    last = null;
  const ack = (needBootstrap) => ({ epoch, seq, needBootstrap });
  return {
    cursor() {
      return { epoch, seq };
    },
    retire() {
      epoch = null;
      seq = 0;
      last = null;
    },
    receiveBootstrap(frame) {
      need(identity(frame.epoch) && safe(frame.seq), 'invalid bootstrap cursor');
      encoded(frame, SKETCH_BOOTSTRAP_BYTES);
      if (frame.epoch === epoch && frame.seq <= seq) return ack(false);
      installBootstrap(frame.display);
      epoch = frame.epoch;
      seq = frame.seq;
      last = null;
      return ack(false);
    },
    receiveBatch(batch) {
      need(
        identity(batch.epoch) &&
          safe(batch.from) &&
          safe(batch.to) &&
          batch.from > 0 &&
          batch.to >= batch.from &&
          Array.isArray(batch.patches) &&
          batch.patches.length === batch.to - batch.from + 1 &&
          batch.patches.length <= 32,
        'invalid batch',
      );
      const json = encoded(batch, SKETCH_BATCH_BYTES);
      if (batch.epoch !== epoch) return ack(true);
      if (batch.to <= seq) {
        if (last && batch.from === last.from && batch.to === last.to)
          need(json === last.json, 'conflicting duplicate batch');
        return ack(false);
      }
      if (batch.from !== seq + 1) return ack(true);
      // applyBatch must validate atomically; a refusal never advances the ACK.
      applyBatch(batch.patches);
      seq = batch.to;
      last = { from: batch.from, to: batch.to, json };
      return ack(false);
    },
  };
}

export function createSketchTransfer({
  maxBytes = SKETCH_BOOTSTRAP_BYTES,
  maxChunk = SKETCH_TRANSFER_BYTES,
} = {}) {
  need(
    safe(maxBytes) &&
      maxBytes <= SKETCH_BOOTSTRAP_BYTES &&
      safe(maxChunk) &&
      maxChunk > 0 &&
      maxChunk <= SKETCH_TRANSFER_BYTES,
    'invalid transfer configuration',
  );
  let active = null;
  return {
    begin({ handle, epoch, total }) {
      need(!active, 'transfer already active');
      need(identity(handle) && identity(epoch) && safe(total) && total <= maxBytes, 'invalid transfer bound');
      active = { handle, epoch, total, offset: 0, bytes: new Uint8Array(total) };
      return { offset: 0, complete: total === 0, ...(total === 0 ? { value: new Uint8Array() } : {}) };
    },
    append(chunk) {
      need(
        active &&
          chunk.handle === active.handle &&
          chunk.epoch === active.epoch &&
          chunk.total === active.total,
        'retired or foreign transfer',
      );
      need(
        safe(chunk.offset) &&
          chunk.bytes instanceof Uint8Array &&
          chunk.bytes.length > 0 &&
          chunk.bytes.length <= maxChunk &&
          chunk.offset + chunk.bytes.length <= active.total,
        'invalid transfer chunk',
      );
      if (chunk.offset < active.offset) {
        need(
          chunk.offset + chunk.bytes.length <= active.offset &&
            chunk.bytes.every((b, i) => b === active.bytes[chunk.offset + i]),
          'conflicting duplicate chunk',
        );
      } else {
        need(chunk.offset === active.offset, 'transfer gap');
        active.bytes.set(chunk.bytes, chunk.offset);
        active.offset += chunk.bytes.length;
      }
      const complete = active.offset === active.total;
      return { offset: active.offset, complete, ...(complete ? { value: active.bytes.slice() } : {}) };
    },
    retire() {
      active = null;
    },
    cursor() {
      return active
        ? { handle: active.handle, epoch: active.epoch, total: active.total, offset: active.offset }
        : null;
    },
  };
}
