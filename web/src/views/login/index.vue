<template>
  <div class="login-wrapper">
    <div class="login-container">
      <div class="login-left">
        <div class="login-info">
          <h1 class="title">AtlHyper</h1>
          <p class="subtitle">Kubernetes Dashboard.</p>
        </div>
      </div>

      <div class="login-right">
        <div class="login-form-wrapper">
          <h2 class="form-title">Welcome AtlHyper</h2>
          <p class="form-subtitle">
            Please enter your account information to manage your cluster
          </p>

          <el-form
            ref="loginForm"
            :model="loginForm"
            :rules="loginRules"
            class="login-form"
            autocomplete="on"
          >
            <!-- <el-form-item prop="username">
              <el-input
                v-model="loginForm.username"
                placeholder="Please enter your username"
                prefix-icon="el-icon-user"
              />
            </el-form-item> -->

            <el-form-item prop="username">
              <el-input
                ref="username"
                v-model="loginForm.username"
                placeholder="Please enter your username"
                prefix-icon="el-icon-user"
              />
            </el-form-item>

            <el-form-item prop="password">
              <el-input
                ref="password"
                v-model="loginForm.password"
                :type="passwordType"
                placeholder="Please enter your password"
                prefix-icon="el-icon-lock"
                @keyup.enter.native="handleLogin"
              >
                <template slot="suffix">
                  <i
                    :class="
                      passwordType === 'password'
                        ? 'el-icon-view'
                        : 'el-icon-view-off'
                    "
                    style="cursor: pointer; color: #999"
                    @click="showPwd"
                  />
                </template>
              </el-input>
            </el-form-item>

            <el-button
              type="primary"
              :loading="loading"
              class="login-button"
              @click="handleLogin"
            >
              Log In
            </el-button>
          </el-form>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { validUsername } from "@/utils/validate";
import SocialSign from "./components/SocialSignin";

export default {
  name: "Login",
  components: { SocialSign },
  data() {
    const validateUsername = (rule, value, callback) => {
      if (!validUsername(value)) {
        callback(new Error("Please enter the correct user name"));
      } else {
        callback();
      }
    };
    const validatePassword = (rule, value, callback) => {
      if (value.length < 5) {
        callback(new Error("The password can not be less than 5 digits"));
      } else {
        callback();
      }
    };
    return {
      loginForm: {
        username: "admin",
        password: "123456",
      },
      loginRules: {
        username: [
          { required: true, trigger: "blur", validator: validateUsername },
        ],
        password: [
          { required: true, trigger: "blur", validator: validatePassword },
        ],
      },
      passwordType: "password",
      capsTooltip: false,
      loading: false,
      showDialog: false,
      redirect: undefined,
      otherQuery: {},
    };
  },
  watch: {
    $route: {
      handler: function (route) {
        const query = route.query;
        if (query) {
          this.redirect = query.redirect;
          this.otherQuery = this.getOtherQuery(query);
        }
      },
      immediate: true,
    },
  },
  created() {
    // window.addEventListener('storage', this.afterQRScan)
  },
  mounted() {
    if (this.loginForm.username === "") {
      this.$refs.username.focus();
    } else if (this.loginForm.password === "") {
      this.$refs.password.focus();
    }
  },
  destroyed() {
    // window.removeEventListener('storage', this.afterQRScan)
  },
  methods: {
    checkCapslock(e) {
      const { key } = e;
      this.capsTooltip = key && key.length === 1 && key >= "A" && key <= "Z";
    },
    showPwd() {
      if (this.passwordType === "password") {
        this.passwordType = "";
      } else {
        this.passwordType = "password";
      }
      this.$nextTick(() => {
        this.$refs.password.focus();
      });
    },
    handleLogin() {
      this.$refs.loginForm.validate((valid) => {
        if (valid) {
          this.loading = true;
          this.$store
            .dispatch("user/login", this.loginForm)
            .then(() => {
              this.$router.push({
                path: this.redirect || "/",
                query: this.otherQuery,
              });
              this.loading = false;
            })
            .catch(() => {
              this.loading = false;
            });
        } else {
          console.log("error submit!!");
          return false;
        }
      });
    },
    getOtherQuery(query) {
      return Object.keys(query).reduce((acc, cur) => {
        if (cur !== "redirect") {
          acc[cur] = query[cur];
        }
        return acc;
      }, {});
    },
  },
};
</script>

<style lang="scss" scoped>
.login-wrapper {
  position: relative;
  height: 100vh;
  width: 100%;
  display: flex;
  justify-content: center;
  align-items: center;
  overflow: hidden;

  &::before {
    content: "";
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: url("~@/assets/img/bg.jpg") no-repeat center center;
    background-size: cover;
    filter: blur(8px);
    z-index: -1;
  }
}

.login-container {
  display: flex;
  width: 1125px;
  height: 625px;
  background: rgba(255, 255, 255, 0.92); // 半透明卡片
  border-radius: 10px;
  overflow: hidden;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.25);
  backdrop-filter: blur(6px);
}

.login-left {
  width: 50%;
  background: linear-gradient(135deg, #7a7fd5, #86a8e7, #91eae4);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  flex-direction: column;
  padding: 30px;
}

.login-info {
  text-align: center;
}

.title {
  font-size: 40px;
  font-weight: bold;
}

.subtitle {
  margin-top: 12px;
  font-size: 20px;
  opacity: 0.9;
}

.login-right {
  width: 50%;
  background: transparent;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px;
}

.login-form-wrapper {
  width: 100%;
}

.form-title {
  font-size: 26px;
  font-weight: bold;
  text-align: left; // ✅ 改为靠左
  margin-bottom: 10px;
}

.form-subtitle {
  font-size: 16px;
  text-align: left; // ✅ 改为靠左
  margin-bottom: 30px;
  color: #666;
}

.login-form ::v-deep(.el-form-item) {
  margin-bottom: 22px;
}

.login-form ::v-deep(.el-input__inner) {
  height: 46px;
  font-size: 16px;
  border-radius: 8px;
  padding-left: 42px !important;
  border: 1px solid #dcdfe6;
  background-color: rgba(255, 255, 255, 0.95);
  transition: all 0.3s ease;
  box-shadow: inset 0 0 0 transparent;

  &:focus {
    border-color: #3ec8f1;
    box-shadow: 0 0 6px rgba(62, 200, 241, 0.5);
  }

  &::placeholder {
    color: #bbb;
    font-size: 15px;
  }
}

.login-form ::v-deep(.el-input__prefix) {
  left: 10px;
  color: #3ec8f1;
  font-size: 18px;
  display: flex;
  align-items: center;
  height: 100%;
}

.login-button {
  width: 100%;
  margin-top: 15px;
  height: 46px;
  font-size: 16px;
  border-radius: 6px;
  background-color: #3ec8f1 !important;
  border: none !important;
  color: white !important;

  &:hover {
    background-color: #06a0e7 !important;
  }
}
</style>
