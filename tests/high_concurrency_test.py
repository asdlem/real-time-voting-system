#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import requests
import time
import random
import json
import threading
import argparse
import statistics
from concurrent.futures import ThreadPoolExecutor
from collections import Counter, defaultdict
import uuid

# 后端API地址
API_URL = "http://localhost:8090/api/polls/{poll_id}/vote"
ADMIN_API_URL = "http://localhost:8090/api/admin/polls/{poll_id}/reset"
CACHE_CLEAN_URL = "http://localhost:8090/api/admin/cache/clean"

# 全局统计变量
stats = {
    "total_requests": 0,
    "successful_requests": 0,
    "rate_limited_requests": 0,
    "duplicate_rejected": 0,
    "other_errors": 0,
    "response_times": [],
    "errors": defaultdict(int)
}

# 线程锁，用于更新统计信息
stats_lock = threading.Lock()

def reset_poll_votes(poll_id, admin_key="admin123"):
    """重置投票的所有票数为0"""
    url = ADMIN_API_URL.format(poll_id=poll_id)
    
    try:
        response = requests.post(
            url, 
            json={"admin_key": admin_key},
            headers={"Content-Type": "application/json"}
        )
        
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

def clean_redis_cache(admin_key="admin123"):
    """清理Redis缓存"""
    try:
        # 发送请求并检查多种可能的字段名
        payload = {
            "admin_key": admin_key, 
            "patterns": ["poll:*", "vote_lock:*"],
            "pattern": "poll:*"  # 尝试另一种可能的字段名
        }
        
        response = requests.post(
            CACHE_CLEAN_URL,
            json=payload,
            headers={"Content-Type": "application/json"}
        )
        
        if response.status_code == 200:
            print("Redis缓存已清理")
            return True
        else:
            print(f"缓存清理失败! 状态码: {response.status_code}")
            print(f"响应: {response.text}")
            return False
    except Exception as e:
        print(f"缓存清理请求出错: {str(e)}")
        return False

def get_poll_details(poll_id):
    """获取投票详情"""
    url = f"http://localhost:8090/api/polls/{poll_id}"
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

def random_ip():
    """生成随机IP地址"""
    return f"{random.randint(1, 255)}.{random.randint(0, 255)}.{random.randint(0, 255)}.{random.randint(0, 255)}"

def random_user_agent():
    """生成随机User-Agent"""
    user_agents = [
        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
        "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15",
        "Mozilla/5.0 (iPhone; CPU iPhone OS 14_4 like Mac OS X) AppleWebKit/605.1.15",
        "Mozilla/5.0 (iPad; CPU OS 14_4 like Mac OS X) AppleWebKit/605.1.15",
        "Mozilla/5.0 (Linux; Android 10; SM-G981B) AppleWebKit/537.36"
    ]
    return random.choice(user_agents)

def send_vote(poll_id, option_id, ip=None, custom_headers=None):
    """发送单个投票请求并返回结果"""
    start_time = time.time()

    url = API_URL.format(poll_id=poll_id)
    payload = {"option_id": option_id}

    headers = {
        "Content-Type": "application/json",
        "X-Forwarded-For": ip or random_ip(),  # 模拟不同IP
        "User-Agent": random_user_agent()      # 模拟不同浏览器
    }
    
    # 添加自定义头信息
    if custom_headers:
        headers.update(custom_headers)

    try:
        response = requests.post(url, json=payload, headers=headers)
        response_time = time.time() - start_time
        
        # 线程安全地更新统计信息
        with stats_lock:
            stats["total_requests"] += 1
            stats["response_times"].append(response_time)
            
            if response.status_code == 200:
                stats["successful_requests"] += 1
                return True, response_time, None
            elif response.status_code == 429:  # 限流响应
                stats["rate_limited_requests"] += 1
                return False, response_time, "RATE_LIMITED"
            elif response.status_code == 400 and "重复" in response.text:
                stats["duplicate_rejected"] += 1
                return False, response_time, "DUPLICATE"
            else:
                stats["other_errors"] += 1
                error_key = f"{response.status_code}: {response.text[:50]}"
                stats["errors"][error_key] += 1
                return False, response_time, f"ERROR: {response.status_code}"
                
    except Exception as e:
        error_msg = str(e)
        with stats_lock:
            stats["total_requests"] += 1
            stats["other_errors"] += 1
            stats["errors"][error_msg] += 1
        return False, 0, error_msg

def test_rate_limiting(poll_id, option_id, num_requests=100, concurrency=10):
    """测试限流器功能：短时间内发送大量来自同一IP的请求"""
    print(f"\n===== 测试限流器 =====")
    print(f"使用同一IP地址发送 {num_requests} 个请求，并发数: {concurrency}")
    
    # 测试前重置投票数据
    reset_poll_votes(poll_id)
    time.sleep(1)  # 等待重置生效
    
    # 重置统计信息
    global stats
    stats = {
        "total_requests": 0,
        "successful_requests": 0,
        "rate_limited_requests": 0,
        "duplicate_rejected": 0,
        "other_errors": 0,
        "response_times": [],
        "errors": defaultdict(int)
    }
    
    same_ip = random_ip()
    
    def worker():
        send_vote(poll_id, option_id, ip=same_ip)
    
    with ThreadPoolExecutor(max_workers=concurrency) as executor:
        # 同时提交所有请求
        futures = [executor.submit(worker) for _ in range(num_requests)]
        # 等待所有请求完成
        for future in futures:
            future.result()
    
    print("\n限流器测试结果:")
    print(f"总请求数: {stats['total_requests']}")
    print(f"成功请求数: {stats['successful_requests']}")
    print(f"被限流请求数: {stats['rate_limited_requests']}")
    print(f"被拒绝的重复请求: {stats['duplicate_rejected']}")
    print(f"其他错误: {stats['other_errors']}")
    
    if stats["response_times"]:
        avg_time = sum(stats["response_times"]) / len(stats["response_times"])
        print(f"平均响应时间: {avg_time:.4f}秒")
    
    if stats["rate_limited_requests"] > 0 or stats["duplicate_rejected"] > 0:
        print("\n✅ 限流器或防重复机制工作正常！检测到被限流或拒绝的请求。")
    else:
        print("\n❌ 限流器和防重复机制可能未正常工作！未检测到任何被限流或拒绝的请求。")
    
    # 显示错误详情
    if stats["errors"]:
        print("\n错误详情:")
        for error, count in stats["errors"].items():
            print(f"  - {error}: {count}次")

def test_duplicate_prevention(poll_id, option_id, num_retries=5, ip_count=20):
    """测试防重复提交机制：多次使用同一IP投票"""
    print(f"\n===== 测试防重复提交机制 =====")
    print(f"使用 {ip_count} 个不同IP，每个IP尝试连续投票 {num_retries} 次")
    
    # 测试前重置投票数据
    reset_poll_votes(poll_id)
    time.sleep(1)  # 等待重置生效
    
    # 重置统计信息
    global stats
    stats = {
        "total_requests": 0,
        "successful_requests": 0,
        "rate_limited_requests": 0,
        "duplicate_rejected": 0,
        "other_errors": 0,
        "response_times": [],
        "errors": defaultdict(int),
        "per_ip_success": {}
    }
    
    # 生成固定的IP列表
    ip_list = [random_ip() for _ in range(ip_count)]
    stats["per_ip_success"] = {ip: 0 for ip in ip_list}
    
    # 首先，每个IP发送一次请求（第一次应该都成功）
    print("阶段1: 每个IP首次投票...")
    for ip in ip_list:
        success, _, _ = send_vote(poll_id, option_id, ip=ip)
        if success:
            stats["per_ip_success"][ip] += 1
    
    # 等待一小段时间
    time.sleep(0.5)
    
    # 然后，每个IP再尝试多次投票（应该被拒绝）
    print(f"阶段2: 每个IP再尝试 {num_retries-1} 次投票...")
    for retry in range(1, num_retries):
        for ip in ip_list:
            success, _, _ = send_vote(poll_id, option_id, ip=ip)
            if success:
                stats["per_ip_success"][ip] += 1
        # 每轮之间稍微等待，让服务器有时间处理
        time.sleep(0.2)
    
    # 分析结果
    ips_with_multiple_success = [ip for ip, count in stats["per_ip_success"].items() if count > 1]
    
    print("\n防重复提交测试结果:")
    print(f"总请求数: {stats['total_requests']}")
    print(f"成功请求数: {stats['successful_requests']}")
    print(f"被限流请求数: {stats['rate_limited_requests']}")
    print(f"被拒绝的重复请求: {stats['duplicate_rejected']}")
    print(f"其他错误: {stats['other_errors']}")
    print(f"成功投票多次的IP数量: {len(ips_with_multiple_success)}/{ip_count}")
    
    # 计算防重复效率
    expected_first_votes = ip_count
    expected_rejected = ip_count * (num_retries - 1)
    actual_rejected = stats["duplicate_rejected"] + stats["rate_limited_requests"]
    rejection_rate = (actual_rejected / expected_rejected) * 100 if expected_rejected > 0 else 0
    
    print(f"防重复有效率: {rejection_rate:.2f}%")
    
    if rejection_rate >= 95:
        print("\n✅ 防重复机制工作非常好! (拒绝率 >= 95%)")
    elif rejection_rate >= 80:
        print("\n⚠️ 防重复机制工作基本正常，但有优化空间 (拒绝率 >= 80%)")
    else:
        print("\n❌ 防重复机制存在问题! (拒绝率 < 80%)")
        if ips_with_multiple_success:
            print(f"问题IP示例: {ips_with_multiple_success[:3]}")
    
    # 如果有防重复机制失效的IP，显示详情
    if ips_with_multiple_success:
        print("\n防重复失效IP详情:")
        for i, ip in enumerate(ips_with_multiple_success[:5]):  # 只显示前5个
            print(f"  - IP {ip}: 成功投票 {stats['per_ip_success'][ip]} 次")
        if len(ips_with_multiple_success) > 5:
            print(f"  - ... 以及其他 {len(ips_with_multiple_success) - 5} 个IP")

def test_mixed_options(poll_id, options, request_count=200, concurrency=50):
    """测试多选项混合投票：随机给不同选项投票"""
    print(f"\n===== 测试多选项混合投票 =====")
    print(f"对 {len(options)} 个选项随机投票，总计 {request_count} 次，并发: {concurrency}")
    
    # 测试前重置投票数据
    reset_poll_votes(poll_id)
    time.sleep(1)  # 等待重置生效
    
    # 重置统计信息
    global stats
    stats = {
        "total_requests": 0,
        "successful_requests": 0,
        "rate_limited_requests": 0,
        "duplicate_rejected": 0,
        "other_errors": 0,
        "response_times": [],
        "errors": defaultdict(int),
        "per_option_success": {opt: 0 for opt in options}
    }
    
    def worker():
        # 随机选择一个选项
        option_id = random.choice(options)
        # 使用随机IP避免被重复检测
        ip = random_ip()
        success, _, _ = send_vote(poll_id, option_id, ip=ip)
        if success:
            with stats_lock:
                stats["per_option_success"][option_id] += 1
    
    with ThreadPoolExecutor(max_workers=concurrency) as executor:
        futures = [executor.submit(worker) for _ in range(request_count)]
        for future in futures:
            future.result()
    
    print("\n多选项混合投票测试结果:")
    print(f"总请求数: {stats['total_requests']}")
    print(f"成功请求数: {stats['successful_requests']}")
    print(f"被限流请求数: {stats['rate_limited_requests']}")
    print(f"被拒绝的重复请求: {stats['duplicate_rejected']}")
    print(f"其他错误: {stats['other_errors']}")
    
    # 显示每个选项的成功投票数
    print("\n每个选项的成功投票数:")
    for option_id, count in stats["per_option_success"].items():
        print(f"  - 选项 {option_id}: {count} 票")
    
    # 验证系统是否在处理不同选项时都有相似的表现
    success_rates = [count / stats["total_requests"] * len(options) for count in stats["per_option_success"].values()]
    min_rate = min(success_rates) if success_rates else 0
    max_rate = max(success_rates) if success_rates else 0
    
    if min_rate > 0 and max_rate / min_rate < 1.5:
        print("\n✅ 各选项处理均衡，未发现明显偏差")
    else:
        print("\n⚠️ 各选项投票成功率不平衡，可能存在偏差或随机性较高")
    
    time.sleep(1)  # 等待所有数据库操作完成
    verify_data_consistency(poll_id, stats["successful_requests"])

def test_ip_reuse_pattern(poll_id, option_ids, ip_pool_size=10, requests_per_ip=3, delay=0.5):
    """测试IP复用模式：模拟少量IP多次使用的场景（例如NAT环境）"""
    print(f"\n===== 测试IP复用模式 =====")
    print(f"使用 {ip_pool_size} 个IP地址池，每个IP使用 {requests_per_ip} 次，间隔 {delay} 秒")
    
    # 测试前重置投票数据
    reset_poll_votes(poll_id)
    time.sleep(1)  # 等待重置生效
    
    # 重置统计信息
    global stats
    stats = {
        "total_requests": 0,
        "successful_requests": 0,
        "rate_limited_requests": 0,
        "duplicate_rejected": 0,
        "other_errors": 0,
        "response_times": [],
        "errors": defaultdict(int),
        "per_ip_success": {}
    }
    
    # 生成IP池
    ip_pool = [random_ip() for _ in range(ip_pool_size)]
    stats["per_ip_success"] = {ip: 0 for ip in ip_pool}
    
    # 按顺序让每个IP投不同的选项
    total_requests = ip_pool_size * requests_per_ip
    request_counter = 0
    
    print("开始测试IP复用模式...")
    
    for cycle in range(requests_per_ip):
        for i, ip in enumerate(ip_pool):
            request_counter += 1
            print(f"请求 {request_counter}/{total_requests}: IP {ip[:10]}... 第 {cycle+1} 次使用")
            
            # 每次使用不同的选项，避免同一IP对同一选项重复投票
            option_index = (i + cycle) % len(option_ids)
            option_id = option_ids[option_index]
            
            success, _, _ = send_vote(poll_id, option_id, ip=ip)
            if success:
                stats["per_ip_success"][ip] += 1
            
            # 适当延迟，模拟真实用户行为
            if request_counter < total_requests:
                time.sleep(delay)
    
    # 分析结果
    successful_ips = [ip for ip, count in stats["per_ip_success"].items() if count > 0]
    full_success_ips = [ip for ip, count in stats["per_ip_success"].items() if count == requests_per_ip]
    
    print("\nIP复用模式测试结果:")
    print(f"总请求数: {stats['total_requests']}")
    print(f"成功请求数: {stats['successful_requests']}")
    print(f"被限流请求数: {stats['rate_limited_requests']}")
    print(f"被拒绝的重复请求: {stats['duplicate_rejected']}")
    print(f"其他错误: {stats['other_errors']}")
    print(f"至少成功一次的IP数量: {len(successful_ips)}/{ip_pool_size}")
    print(f"全部成功的IP数量: {len(full_success_ips)}/{ip_pool_size}")
    
    # 计算IP复用性能
    expected_success = ip_pool_size * requests_per_ip
    ip_reuse_rate = (stats["successful_requests"] / expected_success) * 100 if expected_success > 0 else 0
    
    print(f"IP复用成功率: {ip_reuse_rate:.2f}%")
    
    if ip_reuse_rate >= 80:
        print("\n✅ 系统允许同一IP投票不同选项，IP复用性能良好")
    elif ip_reuse_rate >= 50:
        print("\n⚠️ 系统对IP复用有一定限制，但基本可用")
    else:
        print("\n❌ 系统对IP复用限制严格，可能影响多人共享IP环境的用户")
    
    time.sleep(1)  # 等待所有数据库操作完成
    verify_data_consistency(poll_id, stats["successful_requests"])

def test_ultra_high_concurrency(poll_id, option_id, num_requests=1000, concurrency=300, batch_size=50):
    """测试超高并发性能：使用分批次的方式发送请求以减轻服务器压力"""
    print(f"\n===== 测试超高并发性能（分批次） =====")
    print(f"发送总计 {num_requests} 个请求，并发数: {concurrency}，每批 {batch_size} 个请求")
    
    # 测试前重置投票数据
    reset_poll_votes(poll_id)
    time.sleep(1)  # 等待重置生效
    
    # 重置统计信息
    global stats
    stats = {
        "total_requests": 0,
        "successful_requests": 0,
        "rate_limited_requests": 0,
        "duplicate_rejected": 0,
        "other_errors": 0,
        "response_times": [],
        "errors": defaultdict(int)
    }
    
    start_time = time.time()
    total_batches = (num_requests + batch_size - 1) // batch_size  # 向上取整
    
    for batch in range(total_batches):
        batch_start = time.time()
        current_batch_size = min(batch_size, num_requests - batch * batch_size)
        
        print(f"执行批次 {batch+1}/{total_batches}，本批请求数: {current_batch_size}")
        
        with ThreadPoolExecutor(max_workers=min(concurrency, current_batch_size)) as executor:
            futures = []
            for i in range(current_batch_size):
                # 使用不同的IP确保请求不会被防重复机制阻止
                ip = random_ip()
                futures.append(executor.submit(send_vote, poll_id, option_id, ip=ip))
            
            # 等待当前批次完成
            for future in futures:
                future.result()
        
        batch_time = time.time() - batch_start
        print(f"批次 {batch+1} 完成，耗时: {batch_time:.2f}秒")
        
        # 短暂休息，让服务器有时间处理
        if batch < total_batches - 1:
            time.sleep(0.2)
    
    total_time = time.time() - start_time
    
    print("\n超高并发测试结果:")
    print(f"总请求数: {stats['total_requests']}")
    print(f"成功请求数: {stats['successful_requests']}")
    print(f"被限流请求数: {stats['rate_limited_requests']}")
    print(f"被拒绝的重复请求: {stats['duplicate_rejected']}")
    print(f"其他错误: {stats['other_errors']}")
    print(f"总耗时: {total_time:.2f}秒")
    
    # 计算RPS（每秒请求数）
    rps = stats["total_requests"] / total_time
    print(f"平均RPS: {rps:.2f}请求/秒")
    
    # 计算响应时间统计
    if stats["response_times"]:
        avg_time = sum(stats["response_times"]) / len(stats["response_times"])
        if len(stats["response_times"]) > 1:
            median_time = statistics.median(stats["response_times"])
            p95_time = sorted(stats["response_times"])[int(len(stats["response_times"]) * 0.95)]
            
            print(f"响应时间统计:")
            print(f"  - 平均: {avg_time:.4f}秒")
            print(f"  - 中位数: {median_time:.4f}秒")
            print(f"  - 95百分位: {p95_time:.4f}秒")
    
    # 获取最终投票详情并验证
    time.sleep(1)  # 等待所有数据库操作完成
    verify_data_consistency(poll_id, stats["successful_requests"])

def verify_data_consistency(poll_id, expected_votes):
    """验证数据一致性：检查数据库中的票数是否与成功请求数匹配"""
    # 等待足够长的时间，确保所有异步操作完成
    print(f"等待系统稳定，确保所有投票已被记录...")
    time.sleep(2)  # 增加等待时间，以便数据库写入完成
    
    # 先清理缓存，确保获取最新数据
    try:
        clean_redis_cache("admin123")
    except Exception as e:
        print(f"清理缓存时出错（非致命）: {e}")
    
    # 再次等待以确保缓存清理生效
    time.sleep(0.5)
    
    poll_data = get_poll_details(poll_id)
    if not poll_data:
        print("无法获取投票详情，验证失败")
        return False
    
    print("\n===== 数据一致性验证 =====")
    
    actual_votes = sum(option.get("votes", 0) for option in poll_data.get("options", []))
    
    print(f"预期票数 (成功请求数): {expected_votes}")
    print(f"实际票数 (数据库总票数): {actual_votes}")
    
    # 计算差异
    difference = abs(actual_votes - expected_votes)
    difference_percent = (difference / expected_votes * 100) if expected_votes > 0 else 0
    
    if difference == 0:
        print("\n✅ 数据完全一致！")
        return True
    elif difference_percent <= 5:
        print(f"\n⚠️ 数据基本一致，差异在可接受范围内 (差异: {difference_percent:.2f}%)")
        return True
    else:
        print(f"\n❌ 数据不一致！差异: {difference} 票 ({difference_percent:.2f}%)")
        print("可能原因:")
        print("  1. 高并发下数据库写入冲突")
        print("  2. 部分请求被判定为重复但未被正确统计")
        print("  3. 事务隔离级别问题")
        print("  4. 前一次测试的数据未完全重置")
        
        # 尝试重置数据，为下一次测试做准备
        print("\n正在尝试重置数据以修复不一致...")
        reset_poll_votes(poll_id)
        return False

def run_continuous_test(poll_id, option_id, duration=10, rate=30):
    """持续压力测试：以固定速率持续发送请求"""
    print(f"\n===== 持续压力测试 =====")
    print(f"以 {rate} 请求/秒的速率持续测试 {duration} 秒")
    
    # 测试前重置投票数据
    reset_poll_votes(poll_id)
    time.sleep(1)  # 等待重置生效
    
    # 重置统计信息
    global stats
    stats = {
        "total_requests": 0,
        "successful_requests": 0,
        "rate_limited_requests": 0,
        "duplicate_rejected": 0,
        "other_errors": 0,
        "response_times": [],
        "errors": defaultdict(int)
    }
    
    # 计算请求间隔
    interval = 1.0 / rate if rate > 0 else 0
    
    # 使用线程池控制并发
    max_workers = min(100, rate * 2)  # 控制最大并发线程数
    with ThreadPoolExecutor(max_workers=max_workers) as executor:
        futures = []
        start_time = time.time()
        
        # 持续发送请求直到达到指定时间
        while time.time() - start_time < duration:
            # 使用随机IP避免被判定为重复提交
            ip = random_ip()
            
            # 提交请求到线程池
            futures.append(executor.submit(send_vote, poll_id, option_id, ip=ip))
            
            # 控制请求速率
            next_request_time = start_time + len(futures) * interval
            now = time.time()
            if next_request_time > now:
                time.sleep(next_request_time - now)
        
        # 等待所有已提交的请求完成
        for future in futures:
            future.result()
    
    total_time = time.time() - start_time
    
    print("\n持续压力测试结果:")
    print(f"测试持续时间: {total_time:.2f}秒")
    print(f"总请求数: {stats['total_requests']}")
    print(f"成功请求数: {stats['successful_requests']}")
    print(f"被限流请求数: {stats['rate_limited_requests']}")
    print(f"被拒绝的重复请求: {stats['duplicate_rejected']}")
    print(f"其他错误: {stats['other_errors']}")
    
    # 计算实际RPS
    actual_rps = stats["total_requests"] / total_time
    print(f"实际RPS: {actual_rps:.2f}请求/秒 (目标: {rate}请求/秒)")
    
    # 计算成功率
    success_rate = (stats["successful_requests"] / stats["total_requests"]) * 100 if stats["total_requests"] > 0 else 0
    print(f"请求成功率: {success_rate:.2f}%")
    
    # 响应时间统计
    if stats["response_times"]:
        avg_time = sum(stats["response_times"]) / len(stats["response_times"])
        if len(stats["response_times"]) > 1:
            median_time = statistics.median(stats["response_times"])
            p95_time = sorted(stats["response_times"])[int(len(stats["response_times"]) * 0.95)]
            
            print(f"响应时间统计:")
            print(f"  - 平均: {avg_time:.4f}秒")
            print(f"  - 中位数: {median_time:.4f}秒")
            print(f"  - 95百分位: {p95_time:.4f}秒")
    
    # 数据一致性验证
    time.sleep(1)  # 等待所有数据库操作完成
    verify_data_consistency(poll_id, stats["successful_requests"])

def debug_data_consistency(poll_id, num_attempts=3):
    """调试数据一致性问题：进行少量投票并检查结果"""
    print(f"\n===== 调试数据一致性问题 =====")
    
    # 重置投票数据
    if not reset_poll_votes(poll_id):
        print("警告: 重置投票数据失败，调试终止")
        return
    
    # 再次确认重置成功
    poll_data = get_poll_details(poll_id)
    if poll_data:
        total_votes = sum(option.get("votes", 0) for option in poll_data.get("options", []))
        if total_votes > 0:
            print(f"警告: 重置后仍有 {total_votes} 票")
            return
    else:
        print("无法获取投票数据，调试终止")
        return
    
    print(f"开始进行 {num_attempts} 次单次投票测试...")
    
    option_id = poll_data["options"][0]["id"]
    success_count = 0
    
    # 全局统计变量
    global stats
    stats = {
        "total_requests": 0,
        "successful_requests": 0,
        "rate_limited_requests": 0,
        "duplicate_rejected": 0,
        "other_errors": 0,
        "response_times": [],
        "errors": defaultdict(int)
    }
    
    for i in range(num_attempts):
        print(f"\n测试 {i+1}/{num_attempts}:")
        
        # 使用随机IP
        ip = random_ip()
        print(f"使用IP: {ip}")
        
        # 发送投票请求
        success, response_time, error = send_vote(poll_id, option_id, ip=ip)
        
        if success:
            success_count += 1
            print(f"✅ 请求成功 (响应时间: {response_time:.4f}秒)")
        else:
            print(f"❌ 请求失败: {error}")
        
        # 每次投票后立即检查
        time.sleep(1)  # 等待数据更新
        
        current_poll_data = get_poll_details(poll_id)
        if current_poll_data:
            current_votes = sum(option.get("votes", 0) for option in current_poll_data.get("options", []))
            print(f"当前总票数: {current_votes} (预期: {success_count})")
            
            # 检查是否一致
            if current_votes != success_count:
                print(f"⚠️ 数据不一致！差异: {current_votes - success_count}")
                # 检查每个选项的票数
                for option in current_poll_data.get("options", []):
                    print(f"  - 选项 {option.get('id')}: {option.get('votes')} 票")
            else:
                print("✅ 数据一致")
        else:
            print("⚠️ 无法获取当前投票数据")
        
        # 每次请求之间添加延迟
        if i < num_attempts - 1:
            time.sleep(0.5)
    
    # 最终结果
    print("\n调试结果汇总:")
    print(f"总请求数: {num_attempts}")
    print(f"成功请求数: {success_count}")
    print(f"最终数据库票数: {current_votes}")
    print(f"数据一致性: {'✅ 一致' if current_votes == success_count else '❌ 不一致'}")
    
    # 分析可能的原因
    if current_votes != success_count:
        print("\n可能的不一致原因分析:")
        if current_votes > success_count:
            print("1. 数据库中可能存在之前测试的残留数据")
            print("2. 重置功能未完全清除所有投票记录")
            print("3. 可能存在事务隔离级别导致的重复计数")
        else:
            print("1. 某些成功请求的投票未被记录到数据库")
            print("2. 可能存在数据库写入失败但API返回成功的情况")
            print("3. 缓存与数据库不同步")
        
        # 建议解决方案
        print("\n建议解决方案:")
        print("1. 检查后端重置功能实现，确保完全清除所有投票记录")
        print("2. 考虑在清理投票时使用直接的SQL语句而不是ORM操作")
        print("3. 确保事务正确提交和回滚")
        print("4. 检查Redis缓存清理是否完整")

def main():
    """主函数，运行性能测试"""
    parser = argparse.ArgumentParser(description="实时投票系统高并发性能测试")
    parser.add_argument("--poll-id", type=int, required=True, help="要测试的投票ID")
    parser.add_argument("--option-id", type=int, help="要投票的选项ID (如不指定，将从投票中随机选择)")
    parser.add_argument("--reset", action="store_true", help="测试前重置投票数据")
    parser.add_argument("--clean-cache", action="store_true", help="测试前清理Redis缓存")
    parser.add_argument("--test-type", choices=["all", "ratelimit", "concurrency", "continuous", "ultra", 
                                              "duplicate", "mixed", "ip-reuse", "debug-consistency"], 
                        default="all", help="指定测试类型")
    parser.add_argument("--concurrency", type=int, default=50, help="并发线程数")
    parser.add_argument("--requests", type=int, default=300, help="总请求数")
    parser.add_argument("--batch-size", type=int, default=50, help="批处理大小 (适用于ultra高并发测试)")
    parser.add_argument("--duration", type=int, default=10, help="持续测试时间(秒)")
    parser.add_argument("--rate", type=int, default=30, help="持续测试请求速率(每秒)")
    parser.add_argument("--ip-pool-size", type=int, default=10, help="IP复用测试中的IP池大小")
    parser.add_argument("--requests-per-ip", type=int, default=3, help="IP复用测试中每IP的请求数")
    parser.add_argument("--debug-attempts", type=int, default=5, help="调试一致性检测中的尝试次数")
    parser.add_argument("--no-verify", action="store_true", help="跳过数据一致性验证")
    
    args = parser.parse_args()
    
    # 获取投票详情以获取选项信息
    poll_data = get_poll_details(args.poll_id)
    if not poll_data:
        print(f"无法获取投票ID {args.poll_id} 的详情，测试终止")
        return
    
    # 获取所有选项ID
    all_option_ids = [option.get("id") for option in poll_data.get("options", [])]
    if not all_option_ids:
        print("投票没有选项，测试终止")
        return
    
    # 确定选项ID
    option_id = args.option_id
    if not option_id:
        option_id = all_option_ids[0]  # 默认使用第一个选项
        print(f"未指定选项ID，使用第一个选项: {option_id}")
    
    # 强制先清理缓存和重置投票数据，确保测试开始时状态一致
    # 无论用户是否指定--reset和--clean-cache，都执行这些操作
    print("开始测试前准备工作...")
    
    # 清理缓存
    clean_cache_success = clean_redis_cache()
    if not clean_cache_success:
        print("警告: 缓存清理失败，但测试将继续")
    else:
        print("缓存清理成功")
    
    # 重置投票数据
    reset_success = reset_poll_votes(args.poll_id)
    if not reset_success:
        print("警告: 投票重置失败，但测试将继续")
        if input("是否仍要继续测试? (y/n): ").lower() != 'y':
            print("测试已取消")
            return
    else:
        print("投票数据重置成功")
        
    # 再次获取投票数据，确认重置成功
    poll_data = get_poll_details(args.poll_id)
    total_votes = sum(option.get("votes", 0) for option in poll_data.get("options", []))
    if total_votes > 0:
        print(f"警告: 重置后投票数据仍有 {total_votes} 票，可能影响测试结果")
        if input("是否仍要继续测试? (y/n): ").lower() != 'y':
            print("测试已取消")
            return
    
    # 等待一段时间，确保系统稳定
    print("系统准备就绪，开始测试...")
    time.sleep(1)
    
    # 根据测试类型运行不同的测试
    if args.test_type == "debug-consistency":
        debug_data_consistency(args.poll_id, num_attempts=args.debug_attempts)
        return  # 直接返回，不运行其他测试
        
    # 分开运行各个测试，每次测试都保证数据重置
    if args.test_type in ["all", "ratelimit"]:
        test_rate_limiting(args.poll_id, option_id, num_requests=100, concurrency=args.concurrency)
        if not args.no_verify:
            verify_data_consistency(args.poll_id, stats["successful_requests"])
    
    if args.test_type in ["all", "duplicate"]:
        test_duplicate_prevention(args.poll_id, option_id, num_retries=5, ip_count=20)
        if not args.no_verify:
            verify_data_consistency(args.poll_id, stats["successful_requests"])
    
    if args.test_type in ["all", "mixed"]:
        test_mixed_options(args.poll_id, all_option_ids, request_count=200, concurrency=args.concurrency)
        if not args.no_verify:
            verify_data_consistency(args.poll_id, stats["successful_requests"])
    
    if args.test_type in ["all", "ip-reuse"]:
        test_ip_reuse_pattern(
            args.poll_id, 
            all_option_ids, 
            ip_pool_size=args.ip_pool_size, 
            requests_per_ip=args.requests_per_ip
        )
        if not args.no_verify:
            verify_data_consistency(args.poll_id, stats["successful_requests"])
    
    if args.test_type in ["all", "concurrency"]:
        test_ultra_high_concurrency(
            args.poll_id, 
            option_id, 
            num_requests=args.requests, 
            concurrency=args.concurrency,
            batch_size=args.batch_size
        )
        if not args.no_verify:
            verify_data_consistency(args.poll_id, stats["successful_requests"])
    
    if args.test_type in ["all", "continuous"]:
        run_continuous_test(
            args.poll_id, 
            option_id, 
            duration=args.duration, 
            rate=args.rate
        )
        if not args.no_verify:
            verify_data_consistency(args.poll_id, stats["successful_requests"])
    
    if args.test_type == "ultra":
        # 超高并发测试，使用更多的请求和更高的并发度
        test_ultra_high_concurrency(
            args.poll_id, 
            option_id, 
            num_requests=args.requests, 
            concurrency=args.concurrency * 2,
            batch_size=args.batch_size
        )
        if not args.no_verify:
            verify_data_consistency(args.poll_id, stats["successful_requests"])
            
    # 最后重置投票数据
    print("\n===== 测试结束，重置投票数据 =====")
    reset_poll_votes(args.poll_id)
    print("测试完成!")

if __name__ == "__main__":
    main() 