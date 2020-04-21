package cli

import (
	"fmt"
	"log"
	"os"

	"github.com/99designs/aws-vault/prompt"
	"github.com/99designs/aws-vault/vault"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"gopkg.in/alecthomas/kingpin.v2"
)

type AddCommandInput struct {
	ProfileName string
	Keyring     *vault.CredentialKeyring
	FromEnv     bool
	AddConfig   bool
}

func ConfigureAddCommand(app *kingpin.Application) {
	input := AddCommandInput{}

	cmd := app.Command("add", "Adds credentials, prompts if none provided")
	cmd.Arg("profile", "Name of the profile").
		Required().
		StringVar(&input.ProfileName)

	cmd.Flag("env", "Read the credentials from the environment").
		BoolVar(&input.FromEnv)

	cmd.Flag("add-config", "Add a profile to ~/.aws/config if one doesn't exist").
		Default("true").
		BoolVar(&input.AddConfig)

	cmd.Action(func(c *kingpin.ParseContext) error {
		input.Keyring = &vault.CredentialKeyring{Keyring: keyringImpl}
		AddCommand(app, input)
		return nil
	})
}

func AddCommand(app *kingpin.Application, input AddCommandInput) {
	var accessKeyId, secretKey string

	p, _ := awsConfigFile.ProfileSection(input.ProfileName)
	if p.SourceProfile != "" {
		app.Fatalf("Your profile has a source_profile of %s, adding credentials to %s won't have any effect",
			p.SourceProfile, input.ProfileName)
		return
	}
	if p.IncludeProfile != "" {
		app.Fatalf("Your profile has a include_profile of %s, adding credentials to %s won't have any effect",
			p.IncludeProfile, input.ProfileName)
		return
	} else if p.ParentProfile != "" {
		app.Fatalf("Your profile has a parent_profile of %s, adding credentials to %s won't have any effect",
			p.IncludeProfile, input.ProfileName)
		return
	}

	if input.FromEnv {
		if accessKeyId = os.Getenv("AWS_ACCESS_KEY_ID"); accessKeyId == "" {
			app.Fatalf("Missing value for AWS_ACCESS_KEY_ID")
			return
		}
		if secretKey = os.Getenv("AWS_SECRET_ACCESS_KEY"); secretKey == "" {
			app.Fatalf("Missing value for AWS_SECRET_ACCESS_KEY")
			return
		}
	} else {
		var err error
		if accessKeyId, err = prompt.TerminalPrompt("Enter Access Key ID: "); err != nil {
			app.Fatalf(err.Error())
			return
		}
		if secretKey, err = prompt.TerminalPrompt("Enter Secret Access Key: "); err != nil {
			app.Fatalf(err.Error())
			return
		}
	}

	creds := credentials.Value{AccessKeyID: accessKeyId, SecretAccessKey: secretKey}

	if err := input.Keyring.Set(input.ProfileName, creds); err != nil {
		app.Fatalf(err.Error())
		return
	}

	fmt.Printf("Added credentials to profile %q in vault\n", input.ProfileName)

	sessions := input.Keyring.Sessions()

	if n, _ := sessions.Delete(input.ProfileName); n > 0 {
		fmt.Printf("Deleted %d existing sessions.\n", n)
	}

	if _, hasProfile := awsConfigFile.ProfileSection(input.ProfileName); !hasProfile {
		if input.AddConfig {
			newProfileSection := vault.ProfileSection{
				Name: input.ProfileName,
			}
			log.Printf("Adding profile %s to config at %s", input.ProfileName, awsConfigFile.Path)
			if err := awsConfigFile.Add(newProfileSection); err != nil {
				app.Fatalf("Error adding profile: %#v", err)
			}
		}
	}
}
