import firebase from "firebase"
import "firebase/firestore"

var config = {
    apiKey: "AIzaSyAxQa5YThxvAaSyb8G7w8G_W8HQ2Yl3F00",
    authDomain: "my-project-1529935499185.firebaseapp.com",
    databaseURL: "https://my-project-1529935499185.firebaseio.com",
    projectId: "my-project-1529935499185",
    storageBucket: "my-project-1529935499185.appspot.com",
    messagingSenderId: "81436204908"
  };

class Firebase {
  constructor() {
    firebase.initializeApp(config);
    //this.store = firebase.firestore;
    this.auth = firebase.auth;
  }

  // get polls() {
  //   return this.store().collection('polls');
  // }
}

export default new Firebase();
