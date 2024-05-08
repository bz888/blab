#!/usr/bin/env python3

import speech_recognition as sr

TIMEOUT = 10000  # ms
r = sr.Recognizer()
m = sr.Microphone()


def recognize_speech(mode):
    try:
        print("A moment of silence, please...")
        with m as source:
            r.adjust_for_ambient_noise(source)
        print("Set minimum energy threshold to {}".format(r.energy_threshold))
        while True:
            print("Say something!")
            with m as source:
                audio = r.listen(source)
            print("Got it! Now to recognize it...")
            try:
                # recognize speech using Google Speech Recognition
                if mode == 'google':
                    value = r.recognize_google(audio)
                else:
                    value = r.recognize_sphinx(audio)

                print("You said {}".format(value))
                return value
            except sr.UnknownValueError:
                print("Oops! Didn't catch that")
            except sr.RequestError as e:
                print("Uh oh! Couldn't request results from Google Speech Recognition service; {0}".format(e))
    except KeyboardInterrupt:
        pass


if __name__ == "__main__":
    print(recognize_speech('google'))
