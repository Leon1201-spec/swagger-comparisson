# Swagger comparisson

## Short Description: 
Tool for comparing the swagger endpoints json structure for different API endpoints of your applications. It can be used to see made changes to the application api endpoints with running cronjobs. 

## Usage:
1. As a first step the config.yaml has to be updated with the correct values. These values include the following:
- name: The name of the endpoints
- path: The correct path to the swagger endpoint json format output. This described host will be extended with the path and send a "get" request to download the json file. 
- https: Protocol that is going to be used. When set to "true" https will be used. Otherwise http will be the used protocol.
- hosts: This can be multiple hosts, that will all be used. Each host will publish its own output, not connected to the other described hosts. Each host will create its own folder with a history of the json objects and always compare to the last json file that was requested from that host. 
- slack-webhook: The webhook url for connecting to the correct slack channel where the output is pushed to a specific channel.
- slack-channel: The channel where the output of this code should be published.

2. Implement a pipeline, possible Jenkins (example is in this repository), others need to be added. This pipeline should work as a CronJob (for example every day) to trigger the output of changes made to the defined endpoints. 

3. Trigger the pipeline as a first step. There will be no output, as the code first needs to download an json file of the endpoint as a basic file for comparing to future configurations of the endpoint.

## Description of the code:
The code is setup in multiple functions to get the correct output and see the changes made to the endpoint. 

1. The URL is set up, based on the parameters passed in the configuration file and downloads the json file. If no folder exists with the name of the host, the folder will be created to save the old json files for comparisson.

2. Old and newly downloaded json files are opened and passed into the correct datastructure.

3. A comparisson function is triggered for those two files and a list of changes is returned.

4. The changes are passed to a fuction which is returning the object where the changes are referenced. This function will be triggered again, till no refs are made to the object that included the changes. All of the highest endpoints that are affected are returned in a list.

5. An output is configured with the changes, the name object where the changes have been made, a desciption if something was added, deleted or modified and the changed endpoints.

6. Output is passed to slack and published there.

## Future Implementation and further informations:
At the current point the code is in a beta version and was build a specific company requirements. This code will be updated and further developed in the future. 

Next topics:
- Different CI/CD tools that can be used
- Different output configuration, such as Confluence, Teams, etc. 
- Usage of different paths in one configuration file, to allow different setups of api endpoints in one CronJob
- Possible Setup for CronJob Configuration
- Bugfix endpoint output

If there is interest in this tool and changes are made that could benefit other users, please update the code with a pull request from your own branch with a short description of what has been done.



