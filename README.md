# medium-to-instapaper

Do you have a bunch of reading lists in your medium account and want to add them to Instapaper? then you may find this project helpful.

## Prerequisites
1. You need to have Client ID and Client Secret to interact with Instapaper. Check [this](https://www.instapaper.com/api) article to find out more about that.
2. Go (>= 1.17) installed in your machine

## Steps

1. Get your reading list (and some additional data) from your medium account by exporting them. You can do it by go to the settings > account > Download your information. Medium will send you an email and you will be able to download your medium's data including your reading list
2. From the previous step, you will get a zip file containing your medium data
3. Copy the content of `sample.env` to a new file called `.env` and adjust the values according to your account's setting.
4. Run the command
    ```bash
    make add-to-instapaper
    ```
5. You will see the result of adding operations in `succeed.csv` and `failed.csv` each of them represents the success and failure operations respectively.