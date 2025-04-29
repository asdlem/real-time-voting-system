import requests
import time
import random
import json
import sys
import argparse
import uuid

# 后端API地址 - 确保所有URL使用8090端口
API_URL = "http://localhost:8090/api/polls/{poll_id}/vote"
ADMIN_API_URL = "http://localhost:8090/api/admin/polls/{poll_id}/reset" 

# User-Agent列表，模拟不同的浏览器和设备
USER_AGENTS = [
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Edge/122.0.0.0 Safari/537.36",
    "Mozilla/5.0 (iPhone; CPU iPhone OS 17_4_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
    "Mozilla/5.0 (iPad; CPU OS 17_4_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
    "Mozilla/5.0 (Linux; Android 14; SM-S918B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Mobile Safari/537.36",
    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:124.0) Gecko/20100101 Firefox/124.0",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:124.0) Gecko/20100101 Firefox/124.0",
    "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:124.0) Gecko/20100101 Firefox/124.0"
]

def generate_random_ip():
    """生成随机IP地址"""
    return f"{random.randint(1, 255)}.{random.randint(0, 255)}.{random.randint(0, 255)}.{random.randint(0, 255)}"

def reset_poll_votes(poll_id, admin_key="admin123"):
    """
    重置投票的所有票数为0
    
    Args:
        poll_id: 投票ID
        admin_key: 管理员密钥
    
    Returns:
        True表示重置成功，False表示失败
    """
    url = ADMIN_API_URL.format(poll_id=poll_id)
    
    try:
        # 发送重置请求
        response = requests.post(url, 
                                 json={"admin_key": admin_key},
                                 headers={"Content-Type": "application/json"})
        
        if response.status_code == 200:
            print(f"已成功重置投票ID {poll_id} 的所有票数")
            return True
        else:
            print(f"重置失败! 状态码: {response.status_code}")
            print(f"响应: {response.text}")
            return False
    except Exception as e:
        print(f"重置请求出错: {str(e)}")
        return False

def simulate_vote(poll_id, option_id, user_id=None, ip_address=None, user_agent=None):
    """
    模拟对指定的投票ID进行投票
    
    Args:
        poll_id: 投票ID
        option_id: 选项ID
        user_id: 模拟的用户ID
        ip_address: 用户IP地址
        user_agent: 用户浏览器UA
    
    Returns:
        投票响应数据
    """
    # 确保每个用户都有独特的标识
    if not ip_address:
        ip_address = generate_random_ip()
    if not user_agent:
        user_agent = random.choice(USER_AGENTS)
        
    payload = {"option_ids": [option_id]}  # 更新为新API格式:使用option_ids数组
    url = API_URL.format(poll_id=poll_id)
    
    user_str = f"用户 {user_id}" if user_id else "匿名用户"
    print(f"{user_str} (IP: {ip_address[:10]}...) 投票给选项: {option_id}")
    
    try:
        # 发送投票请求，使用自定义IP和UA
        headers = {
            "Content-Type": "application/json",
            "User-Agent": user_agent,
            "X-Forwarded-For": ip_address,
            "Client-IP": ip_address
        }
        
        response = requests.post(url, json=payload, headers=headers)
        
        # 检查请求是否成功
        if response.status_code == 200:
            print(f"{user_str} 投票成功! 状态码: {response.status_code}")
            try:
                resp_data = response.json()
                # print(f"响应: {json.dumps(resp_data, indent=2, ensure_ascii=False)}")
                return resp_data
            except:
                print(f"响应: {response.text}")
                return None
        else:
            print(f"{user_str} 投票失败: {response.status_code} - {response.text}")
            return None
    except Exception as e:
        print(f"{user_str} 请求出错: {str(e)}")
        return None

def get_poll_details(poll_id):
    """
    获取投票详情
    
    Args:
        poll_id: 投票ID
    
    Returns:
        投票详情数据
    """
    url = f"http://localhost:8090/api/polls/{poll_id}"  # 使用8090端口
    try:
        response = requests.get(url)
        if response.status_code == 200:
            return response.json()
        else:
            print(f"获取投票详情失败! 状态码: {response.status_code}")
            return None
    except Exception as e:
        print(f"获取投票详情出错: {str(e)}")
        return None

def verify_vote_counts(poll_id, expected_counts=None):
    """
    验证投票计数是否符合预期
    
    Args:
        poll_id: 投票ID
        expected_counts: 预期的票数字典 {option_id: count}
    
    Returns:
        验证结果 (成功/失败)
    """
    poll_data = get_poll_details(poll_id)
    if not poll_data:
        return False
    
    print("\n当前投票结果:")
    total_votes = 0
    option_votes = {}
    
    for option in poll_data.get("options", []):
        option_id = option.get("id")
        votes = option.get("votes", 0)
        option_votes[option_id] = votes
        total_votes += votes
        print(f"选项 {option_id}: {votes} 票")
    
    print(f"总票数: {total_votes}")
    
    # 如果有预期票数，检查是否一致
    if expected_counts:
        print("\n预期票数vs实际票数:")
        expected_total = sum(expected_counts.values())
        actual_total = sum(option_votes.values())
        print(f"预期总投票数: {expected_total}")
        print(f"实际总投票数: {actual_total}")
        
        is_correct = True
        for option_id in expected_counts:
            expected = expected_counts[option_id]
            actual = option_votes.get(option_id, 0)
            diff = abs(actual - expected)
            print(f"选项 {option_id}: 预期={expected}, 实际={actual}, 差异={diff}")
            
            if diff > 0:
                is_correct = False
        
        if is_correct:
            print("✅ 验证通过: 所有选项的票数与预期一致!")
        else:
            print("❌ 验证失败: 存在票数与预期不一致!")
        
        return is_correct
    
    return True

def clean_redis_cache(admin_key="admin123", pattern=""):
    """
    清理Redis缓存
    
    Args:
        admin_key: 管理员密钥
        pattern: 要清理的键模式，为空表示清理所有投票相关的缓存
    
    Returns:
        True表示清理成功，False表示失败
    """
    url = "http://localhost:8090/api/admin/cache/clean"  # 使用8090端口
    
    try:
        # 准备请求数据
        payload = {
            "admin_key": admin_key,
            "patterns": ["poll:*", "vote_lock:*"]
        }
        if pattern:
            payload["patterns"].append(pattern)
            
        # 发送清理请求
        response = requests.post(url, 
                                 json=payload,
                                 headers={"Content-Type": "application/json"})
        
        if response.status_code == 200:
            resp_data = response.json()
            print(f"缓存清理成功：删除了 {resp_data.get('deleted_keys', 0)} 个键")
            return True
        else:
            print(f"缓存清理失败! 状态码: {response.status_code}")
            print(f"响应: {response.text}")
            return False
    except Exception as e:
        print(f"缓存清理请求出错: {str(e)}")
        return False

def main():
    """主函数，模拟多个用户连续投票"""
    # 解析命令行参数
    parser = argparse.ArgumentParser(description="多用户投票测试脚本")
    parser.add_argument("--poll-id", type=int, help="要测试的投票ID")
    parser.add_argument("--mode", type=int, choices=[1, 2, 3, 4], help="测试模式: 1=随机, 2=平衡, 3=定向, 4=重复投票测试")
    parser.add_argument("--users", type=int, default=50, help="要模拟的用户数量")
    parser.add_argument("--dup-users", type=int, default=5, help="模式4下，尝试重复投票的用户数量")
    parser.add_argument("--dup-attempts", type=int, default=3, help="模式4下，每个用户尝试重复投票的次数")
    parser.add_argument("--reset", action="store_true", help="测试前重置投票数据")
    parser.add_argument("--clean-cache", action="store_true", help="清理Redis缓存")
    parser.add_argument("--admin-key", default="admin123", help="管理员密钥，用于重置操作")
    args = parser.parse_args()
    
    print("========== 多用户投票测试脚本 ==========")
    
    # 清理Redis缓存
    if args.clean_cache:
        admin_key = args.admin_key or input("请输入管理员密钥: ")
        if not clean_redis_cache(admin_key):
            if input("缓存清理失败，是否继续测试? (y/n): ").lower() != 'y':
                return
    
    # 获取投票ID
    poll_id = args.poll_id
    if not poll_id:
        poll_id = input("请输入要测试的投票ID: ")
        try:
            poll_id = int(poll_id)
        except ValueError:
            print("错误: 投票ID必须是整数!")
            return
    
    # 重置投票数据
    if args.reset or input("是否重置投票数据? (y/n): ").lower() == 'y':
        admin_key = args.admin_key or input("请输入管理员密钥: ")
        if not reset_poll_votes(poll_id, admin_key):
            if input("重置失败，是否继续测试? (y/n): ").lower() != 'y':
                return
    
    # 获取投票详情
    poll_data = get_poll_details(poll_id)
    if not poll_data:
        print(f"无法获取投票ID {poll_id} 的详情，请确认ID是否正确!")
        return
    
    print(f"\n投票主题: {poll_data.get('question', '未知')}")
    
    # 获取所有选项ID
    option_ids = []
    for option in poll_data.get("options", []):
        option_ids.append(option.get("id"))
        print(f"选项 {option.get('id')}: {option.get('text')}")
    
    if not option_ids:
        print("该投票没有选项!")
        return
    
    print(f"\n可用选项ID: {option_ids}")
    
    # 获取测试模式
    mode = args.mode
    if not mode:
        print("\n请选择测试模式:")
        print("1. 随机投票测试 (随机选择选项)")
        print("2. 平衡投票测试 (所有选项获得平均票数)")
        print("3. 定向投票测试 (指定每个选项的票数)")
        print("4. 重复投票测试 (测试对重复投票的检测)")
        mode = int(input("请选择测试模式 (1-4): "))
    
    # 获取用户数量
    num_users = args.users
    
    # 准备用户和选项分配
    users = []
    expected_counts = {option_id: 0 for option_id in option_ids}
    
    print(f"\n准备模拟 {num_users} 个用户进行投票...")
    
    if mode in [1, 2, 3]:  # 正常投票模式
        if mode == 1:  # 随机模式
            # 为每个用户随机分配一个选项
            for i in range(num_users):
                selected_option = random.choice(option_ids)
                user = {
                    "id": i + 1,
                    "ip": generate_random_ip(),
                    "user_agent": random.choice(USER_AGENTS),
                    "option_id": selected_option
                }
                users.append(user)
                expected_counts[selected_option] += 1
        
        elif mode == 2:  # 平衡模式
            # 计算每个选项应获得的票数
            base_votes = num_users // len(option_ids)
            remainder = num_users % len(option_ids)
            
            # 分配票数
            option_votes = {}
            for i, option_id in enumerate(option_ids):
                option_votes[option_id] = base_votes + (1 if i < remainder else 0)
                expected_counts[option_id] = option_votes[option_id]
            
            # 为每个用户分配选项
            user_id = 1
            for option_id, count in option_votes.items():
                for _ in range(count):
                    user = {
                        "id": user_id,
                        "ip": generate_random_ip(),
                        "user_agent": random.choice(USER_AGENTS),
                        "option_id": option_id
                    }
                    users.append(user)
                    user_id += 1
        
        elif mode == 3:  # 定向模式
            user_id = 1
            for option_id in option_ids:
                vote_for_option = int(input(f"请输入要给选项 {option_id} 的票数: "))
                expected_counts[option_id] = vote_for_option
                
                # 创建对应数量的用户
                for _ in range(vote_for_option):
                    user = {
                        "id": user_id,
                        "ip": generate_random_ip(),
                        "user_agent": random.choice(USER_AGENTS),
                        "option_id": option_id
                    }
                    users.append(user)
                    user_id += 1
        
        # 随机打乱用户顺序，模拟真实场景
        random.shuffle(users)
        
        # 执行投票
        print(f"\n开始执行 {len(users)} 个用户的投票，每个用户只投一次票...")
        successful_votes = 0
        start_time = time.time()
        
        # 执行所有投票
        for i, user in enumerate(users):
            print(f"\n===== 用户 {user['id']} 投票 ({i+1}/{len(users)}) =====")
            result = simulate_vote(
                poll_id, 
                user["option_id"], 
                user_id=user["id"], 
                ip_address=user["ip"], 
                user_agent=user["user_agent"]
            )
            
            if result is not None:
                successful_votes += 1
            
            # 每10次投票后验证一次
            if (i + 1) % 10 == 0 or i == len(users) - 1:
                print(f"\n已完成 {i+1} 次投票，进行结果验证...")
                verify_vote_counts(poll_id)
            
            # 适当延迟，模拟正常用户访问间隔
            if i < len(users) - 1:
                delay = random.uniform(0.2, 0.5)  # 随机延迟0.2-0.5秒
                print(f"等待 {delay:.2f} 秒...")
                time.sleep(delay)
        
        # 计算总用时
        total_time = time.time() - start_time
        
        # 最终验证
        print("\n所有投票完成，进行最终验证...")
        final_result = verify_vote_counts(poll_id, expected_counts)
        
        print("\n测试结果摘要:")
        print(f"- 总用时: {total_time:.2f}秒")
        print(f"- 模拟用户数: {len(users)}")
        print(f"- 成功投票数: {successful_votes}")
        print(f"- 预期票数: {expected_counts}")
        print(f"- 最终验证: {'通过' if final_result else '失败'}")
        
    elif mode == 4:  # 重复投票测试模式
        dup_users = args.dup_users
        dup_attempts = args.dup_attempts
        
        if dup_users > num_users:
            dup_users = num_users
            print(f"警告: 重复投票用户数不能大于总用户数，已调整为 {dup_users}")
        
        print(f"\n模式4 - 重复投票测试:")
        print(f"- 总用户数: {num_users}")
        print(f"- 尝试重复投票的用户数: {dup_users}")
        print(f"- 每用户重复尝试次数: {dup_attempts}")
        
        # 创建所有用户
        for i in range(num_users):
            selected_option = random.choice(option_ids)
            user = {
                "id": i + 1,
                "ip": generate_random_ip(),
                "user_agent": random.choice(USER_AGENTS),
                "option_id": selected_option,
                "duplicate": i < dup_users  # 前dup_users个用户为重复投票用户
            }
            users.append(user)
            # 预期计数只包括第一次投票
            expected_counts[selected_option] += 1
        
        # 不再随机打乱用户顺序，而是保持顺序处理
        # 准备所有投票动作，但确保首次投票先执行
        all_votes = []
        
        # 第一轮：所有用户进行首次投票
        for user in users:
            all_votes.append({
                "user": user,
                "attempt": 1,
                "description": "首次投票"
            })
        
        # 第二轮：重复投票用户尝试重复投票
        for user in users:
            if user["duplicate"]:
                for attempt in range(2, dup_attempts + 1):
                    all_votes.append({
                        "user": user,
                        "attempt": attempt,
                        "description": f"第{attempt}次尝试"
                    })
        
        # 执行投票
        print(f"\n开始执行投票测试，首先所有用户进行首次投票，然后重复投票用户进行重复尝试...")
        successful_votes = 0
        duplicate_attempts = 0
        duplicate_successes = 0
        start_time = time.time()
        
        for i, vote in enumerate(all_votes):
            user = vote["user"]
            attempt = vote["attempt"]
            description = vote["description"]
            
            print(f"\n===== 用户 {user['id']} {description} ({i+1}/{len(all_votes)}) =====")
            
            if attempt > 1:
                duplicate_attempts += 1
                print(f"⚠️ 重复投票测试: 用户{user['id']}正在尝试第{attempt}次投票")
            
            result = simulate_vote(
                poll_id, 
                user["option_id"], 
                user_id=user["id"], 
                ip_address=user["ip"], 
                user_agent=user["user_agent"]
            )
            
            if result is not None:
                successful_votes += 1
                if attempt > 1:
                    duplicate_successes += 1
                    print(f"‼️ 警告: 用户{user['id']}的重复投票成功了，这可能表明防重复机制存在问题")
            
            # 每5次投票后验证一次
            if (i + 1) % 5 == 0 or i == len(all_votes) - 1:
                print(f"\n已完成 {i+1} 次投票尝试，进行结果验证...")
                verify_vote_counts(poll_id)
            
            # 适当延迟
            if i < len(all_votes) - 1:
                # 不同用户之间需要更长的延迟
                next_user_id = all_votes[i+1]["user"]["id"]
                if next_user_id != user["id"]:
                    delay = random.uniform(0.3, 0.6)  # 不同用户之间延迟较长
                else:
                    delay = random.uniform(0.5, 1.0)  # 同一用户的重复尝试间隔更长，确保间隔足够
                
                print(f"等待 {delay:.2f} 秒...")
                time.sleep(delay)
        
        # 计算总用时
        total_time = time.time() - start_time
        
        # 最终验证
        print("\n所有投票完成，进行最终验证...")
        final_result = verify_vote_counts(poll_id)
        
        # 防重复机制验证
        duplicate_rejection_rate = 0
        if duplicate_attempts > 0:
            duplicate_rejection_rate = ((duplicate_attempts - duplicate_successes) / duplicate_attempts) * 100
        
        print("\n重复投票测试结果:")
        print(f"- 总用户数: {num_users}")
        print(f"- 尝试重复投票的用户数: {dup_users}")
        print(f"- 重复投票尝试次数: {duplicate_attempts}")
        print(f"- 重复投票成功次数: {duplicate_successes}")
        print(f"- 重复投票拒绝率: {duplicate_rejection_rate:.2f}%")
        
        if duplicate_successes == 0:
            print("✅ 防重复机制工作正常: 所有重复投票都被拒绝!")
        else:
            print(f"❌ 防重复机制存在问题: {duplicate_successes} 次重复投票成功")
        
        print("\n测试结果摘要:")
        print(f"- 总用时: {total_time:.2f}秒")
        print(f"- 总投票尝试: {len(all_votes)}")
        print(f"- 成功投票数: {successful_votes}")
        print(f"- 验证结果: {'通过' if final_result and duplicate_successes == 0 else '失败'}")
    
    else:
        print("无效的模式选择!")
        return
    
    print("\n测试完成!")

if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("\n测试被用户中断")
    except Exception as e:
        print(f"\n测试出现错误: {str(e)}")
        import traceback
        traceback.print_exc()
    finally:
        print("\n脚本已退出")