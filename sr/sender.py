import requests
import logging

# Set up logging
logging.basicConfig(level=logging.DEBUG, format='%(asctime)s - %(levelname)s - %(message)s')


def send_to_api(text):
    url = "http://localhost:8080/process_text"
    data = {'text': text}
    logging.debug("Sending data to API: %s", data)
    response = requests.post(url, json=data)

    if response.status_code == 200:
        logging.debug("Received successful response from API")
        return response.json()
    else:
        logging.error("Error from API: Status Code %d", response.status_code)
        return "Error from API: " + str(response.status_code)


if __name__ == "__main__":
    from recognizer import recognize_speech
    recognized_text = recognize_speech('google')
    logging.info("Test text: %s", recognized_text)
    processed_text = send_to_api(recognized_text)
    logging.info("Processed text: %s", processed_text)
