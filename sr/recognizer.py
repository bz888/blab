#!/usr/bin/env python3

import speech_recognition as sr

TIMEOUT = 10000  # ms


def recognize_speech():
    r = sr.Recognizer()
    with sr.Microphone() as source:
        print("Say something!")
        audio = r.listen(source, TIMEOUT)
    try:
        return r.recognize_sphinx(audio)
    except sr.UnknownValueError:
        return "Unable to understand audio"
    except sr.RequestError as e:
        return "Sphinx error; {0}".format(e)


if __name__ == "__main__":
    recognized_text = recognize_speech()
    print(recognized_text)
