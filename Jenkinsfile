pipeline {
    agent {
        kubernetes {
            cloud 'rancher-desktop'
            yaml '''
apiVersion: v1
kind: Pod
metadata:
  labels:
    app: jenkins-agent
spec:
  containers:
  - name: kaniko-go-builder
    image: gcr.io/kaniko-project/executor:debug
    command:
    - /busybox/cat
    tty: true
'''
        }
    }
    environment {
        GITHUB_REPO = 'https://github.com/kaplunb/kube-webhook-03.git'
        GITHUB_USERNAME = 'kaplunb'
        REGISTRY = 'ghcr.io'
        IMAGE_NAME = "${GITHUB_USERNAME}/validating-webhook"
        IMAGE_TAG = "${env.BUILD_NUMBER}"
        GITHUB_CREDS = credentials('6bcbe156-f69f-461c-ae74-177a09db88ef')
        DOCKER_CONFIG_JSON = "{\"auths\":{\"https://ghcr.io\":{\"username\":\"${GITHUB_USERNAME}\",\"password\":\"${GITHUB_CREDS_PSW}\"}}}"
    }
    stages {
        stage('Checkout') {
            steps {
                git branch: 'github-registry', credentialsId: '6bcbe156-f69f-461c-ae74-177a09db88ef', url: "${env.GITHUB_REPO}"
            }
        }
        stage('Build and Push Image') {
            steps {
                container('kaniko-go-builder') {
                    sh '''
                        echo $DOCKER_CONFIG_JSON > /kaniko/.docker/config.json
                        /kaniko/executor --context $WORKSPACE \
                                        --dockerfile $WORKSPACE/Dockerfile \
                                        --destination $REGISTRY/$IMAGE_NAME:$IMAGE_TAG \
                                        --destination $REGISTRY/$IMAGE_NAME:latest
                    '''
                }
            }
        }
    }
}