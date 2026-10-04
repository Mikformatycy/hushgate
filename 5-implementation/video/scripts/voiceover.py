#!/usr/bin/env python3
"""Generate the launch video's voiceover with ElevenLabs Eleven v4.

    ELEVENLABS_API_KEY=... python3 scripts/voiceover.py [voice ...]

One clip per scene, so each line lands on its scene (start times are in
src/voiceover.ts). Lines follow the Eleven v4 prompting guide: one audio tag
per clause to set delivery, CAPITALS for emphasis, ellipses for pauses.
Clips are written to public/vo/<voice>/NN.mp3.
"""
import json
import os
import pathlib
import sys
import urllib.request

MODEL = "eleven_v4"
VOICES = {
    "tia": "H4quHNsmPxALMhWFVKW6",     # Tia Mirza: rich, premium ad delivery
    "arthur": "C1npRmjB19a6yNkEucvx",  # Arthur: clear, confident American narrator
}
# Lower stability gives the model room for expressive, strongly intonated reads.
SETTINGS = {"stability": 0.3, "similarity_boost": 0.8, "style": 0.6, "use_speaker_boost": True, "speed": 1.0}

LINES = [
    ("01", "[serious, low and intense] Your AI agents read EVERYTHING... [urgent] and send your secrets straight to the model."),
    ("02", "[confident, powerful] Meet HushGate: every secret, MASKED in flight."),
    ("03", "[snappy] Personal data too: IBANs, PESEL, card numbers... all verified."),
    ("04", "[building anticipation] The result? [pause] [triumphant] ZERO leaks."),
    ("05", "[upbeat] One view for EVERY agent."),
    ("06", "[tense, dramatic] A poisoned document hijacks your agent. [powerful] HushGate stops it... in a tenth of a millisecond."),
    ("07", "[wry] Runaway loop? [snappy] Budget hit. Agent halted. [confident] Spend stays CAPPED."),
    ("08", "[firm, commanding] Unregistered devices? Self-hosted models? [booming] BLOCKED."),
    ("09", "[punchy] Nine controls. ONE policy file. Live in a second."),
    ("10", "[assured] And every action... is on record. [warm] Ready for your auditors."),
    ("11", "[proud, powerful] HushGate. [pause] [warm, inspiring] Use AI. [emphatic] Keep your secrets."),
]


def generate(voice: str, key: str, out: pathlib.Path) -> None:
    out.mkdir(parents=True, exist_ok=True)
    for name, text in LINES:
        body = json.dumps({"text": text, "model_id": MODEL, "voice_settings": SETTINGS, "seed": 2026}).encode()
        req = urllib.request.Request(
            f"https://api.elevenlabs.io/v1/text-to-speech/{VOICES[voice]}?output_format=mp3_44100_128",
            data=body, headers={"xi-api-key": key, "Content-Type": "application/json", "Accept": "audio/mpeg"})
        try:
            with urllib.request.urlopen(req, timeout=120) as r:
                (out / f"{name}.mp3").write_bytes(r.read())
        except urllib.error.HTTPError as e:
            sys.exit(f"{voice} {name}: HTTP {e.code} {e.read().decode()[:300]}")
        print(f"{voice}/{name}.mp3")


def main() -> None:
    key = os.environ.get("ELEVENLABS_API_KEY")
    if not key:
        sys.exit("Set ELEVENLABS_API_KEY")
    root = pathlib.Path(__file__).resolve().parent.parent / "public" / "vo"
    for voice in sys.argv[1:] or list(VOICES):
        generate(voice, key, root / voice)


if __name__ == "__main__":
    main()
