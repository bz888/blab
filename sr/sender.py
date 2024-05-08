import requests

URL = 'http://localhost:8080/proces_text'  # TEMP


def send_to_api(text):
    url = URL
    data = {'text': text}
    resp = requests.post(url, json=data)
    if resp.status_code == 200:
        return resp.json()['processedText']
    else:
        return "Error from API"


if __name__ == "__main__":
    from recognizer import recognize_speech
    recognized_text = recognize_speech()
    print(send_to_api(recognized_text))

