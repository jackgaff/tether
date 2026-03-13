export type MicrophoneStream = {
  stop: () => Promise<void>;
};

export async function startMicrophoneStream(
  socket: WebSocket,
  targetSampleRateHz: number
): Promise<MicrophoneStream> {
  if (typeof window === "undefined" || !navigator.mediaDevices?.getUserMedia) {
    throw new Error("Microphone capture is not available in this browser.");
  }

  const stream = await navigator.mediaDevices.getUserMedia({
    audio: {
      channelCount: 1,
      echoCancellation: true,
      noiseSuppression: true,
      autoGainControl: true
    }
  });

  const audioContext = new window.AudioContext();
  const source = audioContext.createMediaStreamSource(stream);
  const processor = audioContext.createScriptProcessor(4096, 1, 1);
  const silentGain = audioContext.createGain();
  silentGain.gain.value = 0;

  source.connect(processor);
  processor.connect(silentGain);
  silentGain.connect(audioContext.destination);

  processor.onaudioprocess = (event) => {
    if (socket.readyState !== WebSocket.OPEN) {
      return;
    }

    const input = event.inputBuffer.getChannelData(0);
    const downsampled = downsampleBuffer(input, audioContext.sampleRate, targetSampleRateHz);
    if (downsampled.length === 0) {
      return;
    }

    socket.send(float32ToPCM16(downsampled));
  };

  return {
    async stop() {
      processor.onaudioprocess = null;
      processor.disconnect();
      source.disconnect();
      silentGain.disconnect();
      stream.getTracks().forEach((track) => track.stop());

      if (audioContext.state !== "closed") {
        await audioContext.close();
      }
    }
  };
}

function downsampleBuffer(
  input: Float32Array,
  sourceSampleRateHz: number,
  targetSampleRateHz: number
): Float32Array {
  if (targetSampleRateHz === sourceSampleRateHz) {
    return input.slice();
  }

  if (targetSampleRateHz > sourceSampleRateHz) {
    throw new Error("Target sample rate must be less than or equal to the source sample rate.");
  }

  const ratio = sourceSampleRateHz / targetSampleRateHz;
  const outputLength = Math.round(input.length / ratio);
  const output = new Float32Array(outputLength);

  let outputIndex = 0;
  let inputIndex = 0;

  while (outputIndex < outputLength) {
    const nextInputIndex = Math.round((outputIndex + 1) * ratio);
    let sum = 0;
    let count = 0;

    for (let index = inputIndex; index < nextInputIndex && index < input.length; index += 1) {
      sum += input[index];
      count += 1;
    }

    output[outputIndex] = count > 0 ? sum / count : 0;
    outputIndex += 1;
    inputIndex = nextInputIndex;
  }

  return output;
}

function float32ToPCM16(input: Float32Array): ArrayBuffer {
  const output = new ArrayBuffer(input.length * 2);
  const view = new DataView(output);

  for (let index = 0; index < input.length; index += 1) {
    const sample = Math.max(-1, Math.min(1, input[index]));
    view.setInt16(index * 2, sample < 0 ? sample * 0x8000 : sample * 0x7fff, true);
  }

  return output;
}
