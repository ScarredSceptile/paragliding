Similar to Assignment 1, we will develop an online service that will allow users to browse information about IGC files. IGC is an international file format for soaring track files that are used by paragliders and gliders. The program will store IGC files metadata in a NoSQL Database (persistent storage). The system will generate events and it will monitor for new events happening from the outside services. The project will make use of Heroku, OpenStack, and AWS Cloud functions. 

The system must be deployed on Heroku, local SkyHigh OpenStack infrastructure, and as a Cloud Function with AWS. The Go source code must be available for inspection by the teaching staff (read-only access is sufficient).

You can re-use Assignment 1 codebase, and substitute the internal in-memory storage with proper DB query subsystem to request information about the IGC tracks. YOU DO NOT NEED to store IGC files in the Database. In fact, you should NOT store them in a Database. All you need to store is the meta information about the IGC track that has been "uploaded". The file itself, after processing, can be discarded. You will keep the associated URL that has been used to upload the track with the track metadata. 

For the development of the IGC processing, you will use an open source IGC library for Go: goigc

The system must be deployed on either Heroku or Google App Engine, and the Go source code must be available for inspection by the teaching staff (read-only access is sufficient).

App will be running on https://immense-mountain-80707.herokuapp.com

═════════════════════════════════════════════════════════════════════════════

How view the app:

All of the application will be under /paragliding/ but not the admin pages

/paragliding/api will get you the information about the app

/paragliding/api/track is where you can POST a track and GET all the ids of the igcs in the app

/paragliding/api/track/`<id>` to get the track of a given id
  
/paragliding/api/track/`<id>`/`<field>` to get the field of the track with the given id.
  
/paragliding/api/ticker/latest to get the last timestamp of the last added track

/paragliding/api/ticker/ to get the 5 first timestamps

/paragliding/api/ticker/`<timestamp>` to get the first 5 timestamps after given timestamp
  
/paragliding/api/webhook/new_track/ is where you can POST a webhook

/paragliding/api/new_track/`<webhook_id>` is where you can GET information about a webhook with given id, or DELETE it
  
═════════════════════════════════════════════════════════════════════════════
  
Available fields for track are:

  pilot
  
  glider
  
  glider_id
  
  track_length
  
  H_date
  
  track_src_url

═════════════════════════════════════════════════════════════════════════════

Format for posting a webhook:
```
{
    "webhookURL": "<WebhookURL>",
    "minTriggerValue": <Number>
}
```
  Where `<WebhookURL` is the URL for your webhook, and `<Number>` is the amount of tracks you want posted before you get a notification about added tracks.

The webhook is formatted for slack, but you can use discord webhook if you add /slack to the end of the url.
  
═════════════════════════════════════════════════════════════════════════════

Clock Trigger:

This is implemented in a way, but due to it shutting down the app, I have disabeled it!
