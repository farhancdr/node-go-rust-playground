import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = {
  stages: [
    { duration: '30s', target: 200 }, // ramp up
    { duration: '1m', target: 100 },  // hold
    { duration: '2m', target: 500 },  // peak load
    { duration: '30s', target: 0 },   // ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<1000'], // 95% < 1s
    http_req_failed: ['rate<0.01'],    // <1% errors
  },
};

export default function () {
  const url = 'https://eu-staging-game-play-service.flarie.com/v1/game/game-coupon-click';

  const payload = JSON.stringify({
    gameId: "f7ab1e7c-4a62-4ccc-ad02-af97e3837637",
    playerId: "d777a0de-ffc3-4f61-b903-e092f5ef7eb6",
    leaderBoardId: "7851de74ea4ee1e3cdc596c489a84212:913875a2cf02076b0d976c32866ce921a0df7a9f8346aad743ad281fdde9a3ad62e6ffe310a7149b6a5c5cc56fb2c6ef70690706f93469a948af6a12faa32aa2",
    organizationId: "044cf3f1-be7c-4020-9158-230262c047aa",
    identifierFormInputType: "UUID",
    identifierValue: "farhan_tx2",
    identifierType: "ADVANCED",
    formResponses: {
      uuid: "farhan_tx2",
      username: "",
      email: "",
      firstName: "",
      lastName: "",
      fullName: "",
      company: "",
      phone: "",
      age: 0,
      gender: "",
      birthday: "",
      address: "",
      businessRegion: "",
      country: "",
      city: "",
      zipCode: "",
      src: "",
      other1: "",
      other2: "",
      other3: "",
      other4: "",
      consent: false,
      consentMandatory: false,
      terms1: false,
      terms2: false,
      terms3: false,
      terms4: false,
      terms5: false,
      param1: "",
      param2: "",
      param3: "",
      param4: "",
      param5: "",
      param6: "",
      param7: "",
      param8: "",
      param9: "",
      param10: ""
    },
    isTermsChecked: true,
    playerMetaInput: {
      ip: "",
      device: "Desktop",
      browser: "Chrome",
      os: "OS X"
    },
    gameName: "puck",
    gameRegion: "eu",
    hostRegion: "eu"
  });

  const params = {
    headers: {
      'accept': 'application/json, text/plain, */*',
      'content-type': 'application/json',
      'customization-key': 'null',
      'origin': 'https://staging-game-eu.flarie.com',
      'referer': 'https://stagi 5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36',
    },
  };

  let res = http.post(url, payload, params);

  check(res, {
    'status is 201': (r) => r.status === 201,
  });

  sleep(1);
}
