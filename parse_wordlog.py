import json
import csv
from datetime import datetime

wrds = {} # track open touches by word

with open("wordlog.csv", "w", newline='') as csvfile:
    csvw = csv.writer(csvfile)
    csvw.writerow(["time", "duration", "word", "source", "choice"])

    with open("wordlog.json", "r") as f:
        for ln in f: # iterate over lines in file
            msg = {}
            try:
                msg = json.loads(ln)
            except json.decoder.JSONDecodeError as err:
                print("cant parse line '" + str(ln) + "': " + str(err))

            if "flavor" in msg:
                if msg["flavor"] == "start_touch":
                    wrd = msg["word"]
                    stmp = msg["time"]
                    if wrd in wrds and wrds[wrd] > 0:
                        pass # ignore vote stutter
                    else:
                        wrds[wrd] = stmp

                elif msg["flavor"] == "end_touch":
                    wrd = msg["word"]
                    stmp = msg["time"]
                    if wrd in wrds:
                        strt = wrds[wrd]
                        if strt > 0:
                            dur = stmp - strt
                            if dur > 200: # suppress stutter
                                if dur < 20000: # suppress very long votes
                                    src = msg["source"]
                                    chc = msg["choice"]
                                    dt = datetime.fromtimestamp(strt/1000.0).strftime('%c')
                                    csvw.writerow([dt, dur, wrd, src, chc])
                                wrds[wrd] = -1
