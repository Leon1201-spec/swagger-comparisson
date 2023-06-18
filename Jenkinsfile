pipeline {
    agent any
    stages{
        stage('BUILD') {
            steps{
                sh "go build main.go"
            }
        }
        stage('RUN'){
            steps{
                sh "./main"
                }
            }
        }
    }