// Copyright 2018, Oracle Corporation and/or its affiliates. All rights reserved.

const fetch = require('node-fetch');

var url = 'http://tacos.wercker.com/api/v1/stores';
var body = {
  "type" : "tacos",
  "name" : "Silly Tacos",
  "address" : "459 Taco Terrace",
  "city" : "Nashua",
  "state" : "NH",
  "zip" : "03062"
};

console.log("creating a store...")
createStore = fetch(url, {
  method: 'PUT',
  body: JSON.stringify(body),
  headers: { 'Content-Type': 'application/json' }
})
.then(res => res.json())
.then(json => json.id)

createStore
.then(storeId => console.log("store with id " + storeId + " created"))


